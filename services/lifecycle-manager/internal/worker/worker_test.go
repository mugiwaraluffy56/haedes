package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExpiredSandboxIsStoppedOnceAndBecomesTerminal(t *testing.T) {
	now := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	repository := newFakeRepository(Sandbox{
		ID:        "sbx_expired",
		State:     StateRunning,
		TaskARN:   "task-expired",
		ExpiresAt: now.Add(-time.Minute),
	})
	compute := &fakeCompute{statuses: map[string]TaskStatus{"task-expired": {Phase: "stopped"}}}
	worker := newTestWorker(t, repository, compute)

	if _, err := worker.ExpireDueSandboxes(context.Background(), now, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := worker.ExpireDueSandboxes(context.Background(), now, 10); err != nil {
		t.Fatal(err)
	}
	if compute.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", compute.stopCalls)
	}
	if repository.sandboxes["sbx_expired"].State != StateStopped {
		t.Fatalf("sandbox state = %s", repository.sandboxes["sbx_expired"].State)
	}
}

func TestOrphanedTaskIsStoppedOnce(t *testing.T) {
	repository := newFakeRepository()
	repository.orphans["task-orphan"] = OrphanTask{ARN: "task-orphan"}
	compute := &fakeCompute{}
	metrics := &fakeMetrics{counts: make(map[string]int)}
	worker, err := New(repository, compute, Config{BatchSize: 10, OperationTimeout: time.Second, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := worker.ReconcileTasks(context.Background(), time.Now().UTC(), 10); err != nil {
		t.Fatal(err)
	}
	if _, err := worker.ReconcileTasks(context.Background(), time.Now().UTC(), 10); err != nil {
		t.Fatal(err)
	}
	if compute.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", compute.stopCalls)
	}
	if len(repository.orphans) != 0 {
		t.Fatalf("orphan task was not marked stopped: %+v", repository.orphans)
	}
	if metrics.counts["sandbox_orphan_total"] != 1 {
		t.Fatalf("orphan metric count = %d, want 1", metrics.counts["sandbox_orphan_total"])
	}
}

func TestStaleStartingSandboxFailsAndStopsItsTask(t *testing.T) {
	now := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	repository := newFakeRepository(Sandbox{
		ID:             "sbx_stale",
		State:          StateStarting,
		TaskARN:        "task-stale",
		LastActivityAt: now.Add(-10 * time.Minute),
	})
	compute := &fakeCompute{statuses: map[string]TaskStatus{"task-stale": {Phase: "running"}}}
	worker := newTestWorker(t, repository, compute)

	if _, err := worker.ReconcileTasks(context.Background(), now, 10); err != nil {
		t.Fatal(err)
	}
	if compute.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", compute.stopCalls)
	}
	if repository.sandboxes["sbx_stale"].State != StateFailed {
		t.Fatalf("sandbox state = %s, want failed", repository.sandboxes["sbx_stale"].State)
	}
}

func TestRunningTaskWithMissingECSRecordFailsSafely(t *testing.T) {
	repository := newFakeRepository(Sandbox{ID: "sbx_missing", State: StateRunning, TaskARN: "task-missing"})
	worker := newTestWorker(t, repository, &fakeCompute{})

	if _, err := worker.ReconcileTasks(context.Background(), time.Now().UTC(), 10); err != nil {
		t.Fatal(err)
	}
	if repository.sandboxes["sbx_missing"].State != StateFailed {
		t.Fatalf("sandbox state = %s, want failed", repository.sandboxes["sbx_missing"].State)
	}
}

func newTestWorker(t *testing.T, repository *fakeRepository, compute *fakeCompute) *Worker {
	t.Helper()
	worker, err := New(repository, compute, Config{
		BatchSize:          10,
		StaleStartingAfter: 5 * time.Minute,
		OperationTimeout:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return worker
}

type fakeRepository struct {
	sandboxes map[string]Sandbox
	orphans   map[string]OrphanTask
}

func newFakeRepository(values ...Sandbox) *fakeRepository {
	sandboxes := make(map[string]Sandbox, len(values))
	for _, value := range values {
		sandboxes[value.ID] = value
	}
	return &fakeRepository{sandboxes: sandboxes, orphans: make(map[string]OrphanTask)}
}

func (repository *fakeRepository) ListExpiredSandboxes(_ context.Context, now time.Time, limit int) ([]Sandbox, error) {
	values := []Sandbox{}
	for _, value := range repository.sandboxes {
		if len(values) == limit {
			break
		}
		if !value.ExpiresAt.IsZero() && !value.ExpiresAt.After(now) && (value.State == StateRunning || value.State == StateStopping) {
			values = append(values, value)
		}
	}
	return values, nil
}

func (repository *fakeRepository) ListReconciliationCandidates(_ context.Context, _ time.Time, limit int) ([]Sandbox, error) {
	values := []Sandbox{}
	for _, value := range repository.sandboxes {
		if len(values) == limit {
			break
		}
		if value.State == StateStarting || value.State == StateRunning || value.State == StateStopping {
			values = append(values, value)
		}
	}
	return values, nil
}

func (repository *fakeRepository) ListOrphanTasks(_ context.Context, _ time.Time, limit int) ([]OrphanTask, error) {
	values := []OrphanTask{}
	for _, value := range repository.orphans {
		if len(values) == limit {
			break
		}
		values = append(values, value)
	}
	return values, nil
}

func (repository *fakeRepository) Transition(_ context.Context, id string, expected, next State, _ string) error {
	value, ok := repository.sandboxes[id]
	if !ok {
		return errors.New("sandbox not found")
	}
	if value.State != expected {
		return ErrStateConflict
	}
	value.State = next
	repository.sandboxes[id] = value
	return nil
}

func (repository *fakeRepository) ClaimStopRequest(_ context.Context, id string) (bool, error) {
	value, ok := repository.sandboxes[id]
	if !ok || value.StopRequestClaimed {
		return false, nil
	}
	value.StopRequestClaimed = true
	repository.sandboxes[id] = value
	return true, nil
}

func (repository *fakeRepository) ClearStopRequest(_ context.Context, id string) error {
	value := repository.sandboxes[id]
	value.StopRequestClaimed = false
	repository.sandboxes[id] = value
	return nil
}

func (repository *fakeRepository) MarkOrphanStopped(_ context.Context, taskARN string) error {
	delete(repository.orphans, taskARN)
	return nil
}

type fakeCompute struct {
	statuses  map[string]TaskStatus
	stopCalls int
}

type fakeMetrics struct {
	counts map[string]int
}

func (metrics *fakeMetrics) Inc(name string, _ map[string]string) {
	metrics.counts[name]++
}

func (compute *fakeCompute) Describe(_ context.Context, taskARN string) (TaskStatus, error) {
	status, ok := compute.statuses[taskARN]
	if !ok {
		return TaskStatus{}, ErrTaskNotFound
	}
	return status, nil
}

func (compute *fakeCompute) Stop(_ context.Context, _ string) error {
	compute.stopCalls++
	return nil
}
