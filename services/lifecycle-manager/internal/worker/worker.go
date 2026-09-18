package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTaskNotFound  = errors.New("lifecycle task not found")
	ErrStateConflict = errors.New("lifecycle state conflict")
)

type State string

const (
	StateProvisioning State = "provisioning"
	StateStarting     State = "starting"
	StateRunning      State = "running"
	StateStopping     State = "stopping"
	StateStopped      State = "stopped"
	StateFailed       State = "failed"
)

type Sandbox struct {
	ID                 string
	State              State
	TaskARN            string
	StopRequestClaimed bool
	ExpiresAt          time.Time
	LastActivityAt     time.Time
}

type OrphanTask struct {
	ARN                string
	StopRequestClaimed bool
}

type TaskStatus struct {
	Phase  string
	Reason string
}

// Repository is the persistence boundary owned by the control plane. Stop
// claims must be atomic so multiple worker instances converge without issuing
// duplicate stop side effects.
type Repository interface {
	ListExpiredSandboxes(ctx context.Context, now time.Time, limit int) ([]Sandbox, error)
	ListReconciliationCandidates(ctx context.Context, now time.Time, limit int) ([]Sandbox, error)
	ListOrphanTasks(ctx context.Context, now time.Time, limit int) ([]OrphanTask, error)
	Transition(ctx context.Context, id string, expected, next State, reason string) error
	ClaimStopRequest(ctx context.Context, id string) (bool, error)
	ClearStopRequest(ctx context.Context, id string) error
	MarkOrphanStopped(ctx context.Context, taskARN string) error
}

type Compute interface {
	Describe(ctx context.Context, taskARN string) (TaskStatus, error)
	Stop(ctx context.Context, taskARN string) error
}

type LifecycleWorker interface {
	ExpireDueSandboxes(ctx context.Context, now time.Time, batchSize int) (int, error)
	ReconcileTasks(ctx context.Context, now time.Time, batchSize int) (int, error)
}

type Config struct {
	BatchSize          int
	StaleStartingAfter time.Duration
	OperationTimeout   time.Duration
}

type RunStats struct {
	Expired        int
	Reconciled     int
	OrphansCleaned int
}

type Worker struct {
	repository Repository
	compute    Compute
	config     Config
}

var _ LifecycleWorker = (*Worker)(nil)

func New(repository Repository, compute Compute, config Config) (*Worker, error) {
	if repository == nil || compute == nil {
		return nil, errors.New("lifecycle repository and compute ports are required")
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.StaleStartingAfter == 0 {
		config.StaleStartingAfter = 5 * time.Minute
	}
	if config.OperationTimeout == 0 {
		config.OperationTimeout = 30 * time.Second
	}
	if config.BatchSize < 1 || config.StaleStartingAfter <= 0 || config.OperationTimeout <= 0 {
		return nil, errors.New("lifecycle worker bounds are invalid")
	}
	return &Worker{repository: repository, compute: compute, config: config}, nil
}

func (worker *Worker) ExpireDueSandboxes(ctx context.Context, now time.Time, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = worker.config.BatchSize
	}
	due, err := worker.repository.ListExpiredSandboxes(ctx, now, batchSize)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, sandbox := range due {
		if err := worker.expireSandbox(ctx, sandbox); err != nil {
			return processed, fmt.Errorf("expire sandbox %s: %w", sandbox.ID, err)
		}
		processed++
	}
	return processed, nil
}

func (worker *Worker) ReconcileTasks(ctx context.Context, now time.Time, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = worker.config.BatchSize
	}
	candidates, err := worker.repository.ListReconciliationCandidates(ctx, now, batchSize)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, sandbox := range candidates {
		if err := worker.reconcileSandbox(ctx, now, sandbox); err != nil {
			return processed, fmt.Errorf("reconcile sandbox %s: %w", sandbox.ID, err)
		}
		processed++
	}

	orphans, err := worker.repository.ListOrphanTasks(ctx, now, batchSize)
	if err != nil {
		return processed, err
	}
	for _, orphan := range orphans {
		if orphan.StopRequestClaimed {
			continue
		}
		claimed, err := worker.claimOrphan(ctx, orphan.ARN)
		if err != nil {
			return processed, err
		}
		if claimed {
			processed++
		}
	}
	return processed, nil
}

func (worker *Worker) RunOnce(ctx context.Context, now time.Time) (RunStats, error) {
	expired, err := worker.ExpireDueSandboxes(ctx, now, worker.config.BatchSize)
	if err != nil {
		return RunStats{Expired: expired}, err
	}
	reconciled, err := worker.ReconcileTasks(ctx, now, worker.config.BatchSize)
	if err != nil {
		return RunStats{Expired: expired, Reconciled: reconciled}, err
	}
	return RunStats{Expired: expired, Reconciled: reconciled}, nil
}

func (worker *Worker) expireSandbox(ctx context.Context, sandbox Sandbox) error {
	switch sandbox.State {
	case StateRunning:
		if err := worker.transition(ctx, sandbox.ID, StateRunning, StateStopping, "sandbox expired"); err != nil {
			return err
		}
		sandbox.State = StateStopping
	case StateStopping:
	default:
		return nil
	}
	if err := worker.stopOnce(ctx, sandbox); err != nil {
		return err
	}
	return worker.finishStopped(ctx, sandbox)
}

func (worker *Worker) reconcileSandbox(ctx context.Context, now time.Time, sandbox Sandbox) error {
	if sandbox.State == StateStarting && isStale(sandbox, now, worker.config.StaleStartingAfter) {
		if err := worker.stopOnce(ctx, sandbox); err != nil {
			return err
		}
		return worker.transition(ctx, sandbox.ID, StateStarting, StateFailed, "sandbox startup deadline expired")
	}
	if sandbox.TaskARN == "" {
		return nil
	}

	status, err := worker.describe(ctx, sandbox.TaskARN)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			if sandbox.State == StateStopping {
				return worker.transition(ctx, sandbox.ID, StateStopping, StateStopped, "task is absent")
			}
			return worker.transition(ctx, sandbox.ID, sandbox.State, StateFailed, "task is absent")
		}
		return err
	}

	switch {
	case sandbox.State == StateStarting && strings.EqualFold(status.Phase, "running"):
		return worker.transition(ctx, sandbox.ID, StateStarting, StateRunning, "runtime task is running")
	case sandbox.State == StateStopping && strings.EqualFold(status.Phase, "stopped"):
		return worker.transition(ctx, sandbox.ID, StateStopping, StateStopped, "task stopped")
	case sandbox.State == StateRunning && strings.EqualFold(status.Phase, "stopped"):
		return worker.transition(ctx, sandbox.ID, StateRunning, StateFailed, "task stopped unexpectedly")
	}
	return nil
}

func (worker *Worker) stopOnce(ctx context.Context, sandbox Sandbox) error {
	if sandbox.TaskARN == "" || sandbox.StopRequestClaimed {
		return nil
	}
	operationContext, cancel := context.WithTimeout(ctx, worker.config.OperationTimeout)
	defer cancel()
	claimed, err := worker.repository.ClaimStopRequest(operationContext, sandbox.ID)
	if err != nil || !claimed {
		return err
	}
	if err := worker.compute.Stop(operationContext, sandbox.TaskARN); err != nil && !errors.Is(err, ErrTaskNotFound) {
		_ = worker.repository.ClearStopRequest(context.Background(), sandbox.ID)
		return err
	}
	return nil
}

func (worker *Worker) finishStopped(ctx context.Context, sandbox Sandbox) error {
	if sandbox.TaskARN == "" {
		return worker.transition(ctx, sandbox.ID, StateStopping, StateStopped, "expired sandbox had no task")
	}
	status, err := worker.describe(ctx, sandbox.TaskARN)
	if err != nil && !errors.Is(err, ErrTaskNotFound) {
		return err
	}
	if errors.Is(err, ErrTaskNotFound) || strings.EqualFold(status.Phase, "stopped") {
		return worker.transition(ctx, sandbox.ID, StateStopping, StateStopped, "expired task stopped")
	}
	return nil
}

func (worker *Worker) describe(ctx context.Context, taskARN string) (TaskStatus, error) {
	operationContext, cancel := context.WithTimeout(ctx, worker.config.OperationTimeout)
	defer cancel()
	return worker.compute.Describe(operationContext, taskARN)
}

func (worker *Worker) claimOrphan(ctx context.Context, taskARN string) (bool, error) {
	if taskARN == "" {
		return false, nil
	}
	operationContext, cancel := context.WithTimeout(ctx, worker.config.OperationTimeout)
	defer cancel()
	if err := worker.compute.Stop(operationContext, taskARN); err != nil && !errors.Is(err, ErrTaskNotFound) {
		return false, err
	}
	return true, worker.repository.MarkOrphanStopped(operationContext, taskARN)
}

func (worker *Worker) transition(ctx context.Context, id string, expected, next State, reason string) error {
	operationContext, cancel := context.WithTimeout(ctx, worker.config.OperationTimeout)
	defer cancel()
	err := worker.repository.Transition(operationContext, id, expected, next, reason)
	if errors.Is(err, ErrStateConflict) {
		return nil
	}
	return err
}

func isStale(sandbox Sandbox, now time.Time, threshold time.Duration) bool {
	return !sandbox.LastActivityAt.IsZero() && now.Sub(sandbox.LastActivityAt) >= threshold
}
