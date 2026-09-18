package fakes

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

var (
	_ sandbox.SandboxRepository  = (*Repository)(nil)
	_ sandbox.ComputeProvisioner = (*Compute)(nil)
	_ sandbox.RuntimeClient      = (*Runtime)(nil)
	_ sandbox.SnapshotRepository = (*SnapshotRepository)(nil)
	_ sandbox.SnapshotStore      = (*ObjectStore)(nil)
)

func TestRepositoryIsDeterministicAndReturnsMissingSandbox(t *testing.T) {
	repository := NewRepository()
	first := sandbox.Sandbox{ID: "sbx_002", OwnerID: "owner", State: sandbox.StateRequested}
	second := sandbox.Sandbox{ID: "sbx_001", OwnerID: "owner", State: sandbox.StateRequested}
	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	page, err := repository.List(context.Background(), "owner", "", 1)
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "sbx_001" || page.NextCursor != "1" {
		t.Fatalf("unexpected first page: %+v, error: %v", page, err)
	}
	page, err = repository.List(context.Background(), "owner", page.NextCursor, 1)
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != "sbx_002" {
		t.Fatalf("unexpected second page: %+v, error: %v", page, err)
	}
	if _, err := repository.Get(context.Background(), "sbx_missing"); !errors.Is(err, sandbox.ErrNotFound) {
		t.Fatalf("missing sandbox error = %v", err)
	}
}

func TestServiceUsesFakeComputeAndRuntimeEvents(t *testing.T) {
	clock := NewClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	compute := NewCompute(clock)
	compute.StartTask = sandbox.TaskRef{ARN: "task-fixed", Endpoint: "http://runtime/fixed", RuntimeToken: "token-fixed"}
	runtime := NewRuntime(clock)
	runtime.Register("sbx_001", compute.StartTask)
	repository := NewRepository()
	service := sandbox.NewService(sandbox.Dependencies{
		Repository: repository,
		Compute:    compute,
		Runtime:    runtime,
		IDs:        testIDs{},
		Clock:      clock,
	})

	created, err := service.Create(context.Background(), "owner", sandbox.SandboxConfig{MaxLifetime: 10 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if created.ExpiresAt != clock.Now().Add(10*time.Minute) {
		t.Fatalf("expiry = %s", created.ExpiresAt)
	}
	result, err := service.Execute(context.Background(), "owner", created.ID, sandbox.CommandRequest{Command: "printf ok"})
	if err != nil || result.ID != "cmd_001" {
		t.Fatalf("command result = %+v, error = %v", result, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, err := service.SubscribeEvents(ctx, "owner", created.ID, result.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"started", "completed"} {
		select {
		case event := <-events:
			if event.Type != expected {
				t.Fatalf("event type = %q, want %q", event.Type, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s event", expected)
		}
	}
	if len(compute.Events) != 1 || compute.Events[0].Kind != "start" {
		t.Fatalf("compute events = %+v", compute.Events)
	}
	if len(runtime.RuntimeEvents) < 2 || runtime.RuntimeEvents[0].Kind != "ready" || runtime.RuntimeEvents[1].Kind != "command" {
		t.Fatalf("runtime events = %+v", runtime.RuntimeEvents)
	}
}

func TestReadinessFailureStopsFakeTaskAndRecordsEvents(t *testing.T) {
	clock := NewClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	compute := NewCompute(clock)
	compute.StartTask = sandbox.TaskRef{ARN: "task-fixed", Endpoint: "http://runtime/fixed", RuntimeToken: "token-fixed"}
	runtime := NewRuntime(clock)
	runtime.ReadyError = errors.New("not ready")
	runtime.Register("sbx_001", compute.StartTask)
	repository := NewRepository()
	service := sandbox.NewService(sandbox.Dependencies{Repository: repository, Compute: compute, Runtime: runtime, IDs: testIDs{}, Clock: clock})
	if _, err := service.Create(context.Background(), "owner", sandbox.SandboxConfig{MaxLifetime: time.Minute}); err == nil {
		t.Fatal("expected readiness failure")
	}
	if len(compute.Events) != 2 || compute.Events[0].Kind != "start" || compute.Events[1].Kind != "stop" {
		t.Fatalf("compute events = %+v", compute.Events)
	}
	failed, err := repository.Get(context.Background(), "sbx_001")
	if err != nil || failed.State != sandbox.StateFailed {
		t.Fatalf("failed sandbox = %+v, error = %v", failed, err)
	}
}

func TestProvisioningFailureIsObservableWithoutAStopCall(t *testing.T) {
	clock := NewClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	compute := NewCompute(clock)
	compute.StartError = errors.New("provisioning failed")
	service := sandbox.NewService(sandbox.Dependencies{
		Repository: NewRepository(),
		Compute:    compute,
		Runtime:    NewRuntime(clock),
		IDs:        testIDs{},
		Clock:      clock,
	})
	if _, err := service.Create(context.Background(), "owner", sandbox.SandboxConfig{MaxLifetime: time.Minute}); err == nil {
		t.Fatal("expected provisioning failure")
	}
	if len(compute.Events) != 1 || compute.Events[0].Kind != "start" {
		t.Fatalf("compute events = %+v", compute.Events)
	}
}

func TestRuntimeFilesystemSnapshotAndRestore(t *testing.T) {
	clock := NewClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	runtime := NewRuntime(clock)
	task := sandbox.TaskRef{Endpoint: "http://runtime/fixed", RuntimeToken: "token-fixed"}
	runtime.Register("sbx_001", task)
	if err := runtime.WriteFile(context.Background(), task.Endpoint, task.RuntimeToken, "hello.txt", strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	archive, info, err := runtime.ExportWorkspace(context.Background(), task.Endpoint, task.RuntimeToken)
	if err != nil {
		t.Fatal(err)
	}
	store := NewObjectStore()
	stored, err := store.Put(context.Background(), "snp_001", archive, info)
	if err != nil || stored.SHA256 != info.SHA256 {
		t.Fatalf("stored info = %+v, error = %v", stored, err)
	}
	opened, _, err := store.Open(context.Background(), "snp_001")
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if err := runtime.DeleteFile(context.Background(), task.Endpoint, task.RuntimeToken, "hello.txt"); err != nil {
		t.Fatal(err)
	}
	if err := runtime.RestoreWorkspace(context.Background(), task.Endpoint, task.RuntimeToken, opened); err != nil {
		t.Fatal(err)
	}
	file, err := runtime.ReadFile(context.Background(), task.Endpoint, task.RuntimeToken, "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	body, _ := io.ReadAll(file)
	if string(body) != "hello" {
		t.Fatalf("restored file = %q", body)
	}
}

type testIDs struct{}

func (testIDs) NewSandboxID() sandbox.SandboxID   { return "sbx_001" }
func (testIDs) NewCommandID() sandbox.CommandID   { return "cmd_ignored" }
func (testIDs) NewSnapshotID() sandbox.SnapshotID { return "snp_001" }
func (testIDs) NewRuntimeToken() string           { return "token-fixed" }
