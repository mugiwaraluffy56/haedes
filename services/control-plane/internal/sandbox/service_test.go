package sandbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type fixedIDs struct{ next int }

func (ids *fixedIDs) NewSandboxID() SandboxID {
	ids.next++
	return SandboxID("sbx_test")
}
func (ids *fixedIDs) NewCommandID() CommandID   { return "cmd_test" }
func (ids *fixedIDs) NewSnapshotID() SnapshotID { return "snp_test" }
func (ids *fixedIDs) NewRuntimeToken() string   { return "runtime-token" }

type memoryRepository struct {
	sandboxes map[SandboxID]Sandbox
	updates   []string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{sandboxes: map[SandboxID]Sandbox{}}
}

func (repository *memoryRepository) Create(_ context.Context, sandbox Sandbox) error {
	repository.sandboxes[sandbox.ID] = sandbox
	repository.updates = append(repository.updates, string(sandbox.State))
	return nil
}
func (repository *memoryRepository) Get(_ context.Context, id SandboxID) (Sandbox, error) {
	sandbox, ok := repository.sandboxes[id]
	if !ok {
		return Sandbox{}, ErrNotFound
	}
	return sandbox, nil
}
func (repository *memoryRepository) List(_ context.Context, ownerID, _ string, _ int) (Page[Sandbox], error) {
	page := Page[Sandbox]{}
	for _, sandbox := range repository.sandboxes {
		if sandbox.OwnerID == ownerID {
			page.Items = append(page.Items, sandbox)
		}
	}
	return page, nil
}
func (repository *memoryRepository) UpdateState(_ context.Context, id SandboxID, expected, next State, _ string) error {
	sandbox := repository.sandboxes[id]
	if sandbox.State != expected {
		return errors.New("state conflict")
	}
	sandbox.State = next
	repository.sandboxes[id] = sandbox
	repository.updates = append(repository.updates, string(next))
	return nil
}
func (repository *memoryRepository) SaveTaskEndpoint(_ context.Context, id SandboxID, task TaskRef) error {
	sandbox := repository.sandboxes[id]
	sandbox.Task = &task
	repository.sandboxes[id] = sandbox
	return nil
}

type fakeCompute struct {
	task       TaskRef
	startError error
	stopCalls  []string
}

func (compute *fakeCompute) Start(_ context.Context, _ ProvisionRequest) (TaskRef, error) {
	return compute.task, compute.startError
}
func (compute *fakeCompute) Stop(_ context.Context, taskARN string) error {
	compute.stopCalls = append(compute.stopCalls, taskARN)
	return nil
}
func (compute *fakeCompute) Describe(_ context.Context, taskARN string) (TaskStatus, error) {
	return TaskStatus{ARN: taskARN}, nil
}

type fakeRuntime struct {
	readyError  error
	startCalls  int
	lastRequest CommandRequest
}

func (runtime *fakeRuntime) WaitReady(_ context.Context, _, _ string) error {
	return runtime.readyError
}
func (runtime *fakeRuntime) StartCommand(_ context.Context, _, _ string, request CommandRequest) (CommandResult, error) {
	runtime.startCalls++
	runtime.lastRequest = request
	return CommandResult{ID: "cmd_test", SandboxID: "sbx_test", Command: request.Command}, nil
}
func (runtime *fakeRuntime) Events(_ context.Context, _, _ string, _ CommandID, _ string) (<-chan CommandEvent, error) {
	return make(chan CommandEvent), nil
}
func (runtime *fakeRuntime) ReadFile(_ context.Context, _, _, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (runtime *fakeRuntime) WriteFile(_ context.Context, _, _, _ string, _ io.Reader) error {
	return nil
}
func (runtime *fakeRuntime) ListFiles(_ context.Context, _, _, _ string) ([]FileEntry, error) {
	return nil, nil
}
func (runtime *fakeRuntime) DeleteFile(_ context.Context, _, _, _ string) error { return nil }
func (runtime *fakeRuntime) ExportWorkspace(_ context.Context, _, _ string) (io.ReadCloser, ArchiveInfo, error) {
	return io.NopCloser(bytes.NewReader(nil)), ArchiveInfo{}, nil
}
func (runtime *fakeRuntime) RestoreWorkspace(_ context.Context, _, _ string, _ io.Reader) error {
	return nil
}

func testService(repository *memoryRepository, compute *fakeCompute, runtime *fakeRuntime) *Service {
	return NewService(Dependencies{
		Repository: repository,
		Compute:    compute,
		Runtime:    runtime,
		IDs:        &fixedIDs{},
		Clock:      fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	})
}

func testConfig() SandboxConfig {
	return SandboxConfig{Image: "haedes:test", MaxLifetime: time.Hour, DefaultCommandTimeout: 10 * time.Second}
}

func TestCreatePersistsBeforeProvisioningAndWaitsForRuntime(t *testing.T) {
	repository := newMemoryRepository()
	compute := &fakeCompute{task: TaskRef{ARN: "task-1", Endpoint: "http://runtime", RuntimeToken: "runtime-token"}}
	runtime := &fakeRuntime{}
	created, err := testService(repository, compute, runtime).Create(context.Background(), "owner-1", testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if created.State != StateRunning {
		t.Fatalf("state = %q, want running", created.State)
	}
	if got := repository.updates; len(got) < 4 || got[0] != "requested" || got[1] != "provisioning" || got[2] != "starting" || got[3] != "running" {
		t.Fatalf("state persistence order = %v", got)
	}
}

func TestCreateStopsTaskWhenRuntimeReadinessFails(t *testing.T) {
	repository := newMemoryRepository()
	compute := &fakeCompute{task: TaskRef{ARN: "task-1", Endpoint: "http://runtime", RuntimeToken: "runtime-token"}}
	runtime := &fakeRuntime{readyError: errors.New("runtime unavailable")}
	_, err := testService(repository, compute, runtime).Create(context.Background(), "owner-1", testConfig())
	if err == nil || len(compute.stopCalls) != 1 {
		t.Fatalf("Create() error = %v, stop calls = %v", err, compute.stopCalls)
	}
	sandbox := repository.sandboxes["sbx_test"]
	if sandbox.State != StateFailed {
		t.Fatalf("persisted state = %q, want failed", sandbox.State)
	}
}

func TestExecuteRoutesOnlyRunningOwnedSandboxes(t *testing.T) {
	repository := newMemoryRepository()
	compute := &fakeCompute{task: TaskRef{ARN: "task-1", Endpoint: "http://runtime", RuntimeToken: "runtime-token"}}
	runtime := &fakeRuntime{}
	service := testService(repository, compute, runtime)
	if _, err := service.Create(context.Background(), "owner-1", testConfig()); err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(context.Background(), "owner-1", "sbx_test", CommandRequest{Command: "printf ok"})
	if err != nil || result.Command != "printf ok" || runtime.startCalls != 1 {
		t.Fatalf("Execute() result = %+v, error = %v, calls = %d", result, err, runtime.startCalls)
	}
	if _, err := service.Execute(context.Background(), "other-owner", "sbx_test", CommandRequest{Command: "bad"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner Execute() error = %v", err)
	}
}

func TestDestroyIsIdempotentAcrossCleanupStates(t *testing.T) {
	repository := newMemoryRepository()
	compute := &fakeCompute{task: TaskRef{ARN: "task-1", Endpoint: "http://runtime", RuntimeToken: "runtime-token"}}
	service := testService(repository, compute, &fakeRuntime{})
	if _, err := service.Create(context.Background(), "owner-1", testConfig()); err != nil {
		t.Fatal(err)
	}

	stopping, err := service.Destroy(context.Background(), "owner-1", "sbx_test")
	if err != nil {
		t.Fatal(err)
	}
	if stopping.State != StateStopping || len(compute.stopCalls) != 1 {
		t.Fatalf("first destroy = %+v, stop calls = %v", stopping, compute.stopCalls)
	}

	repository.sandboxes["sbx_test"] = Sandbox{ID: "sbx_test", OwnerID: "owner-1", State: StateStopped}
	destroyed, err := service.Destroy(context.Background(), "owner-1", "sbx_test")
	if err != nil || destroyed.State != StateDestroyed {
		t.Fatalf("cleanup destroy = %+v, error = %v", destroyed, err)
	}
	repeated, err := service.Destroy(context.Background(), "owner-1", "sbx_test")
	if err != nil || repeated.State != StateDestroyed {
		t.Fatalf("repeated destroy = %+v, error = %v", repeated, err)
	}
}
