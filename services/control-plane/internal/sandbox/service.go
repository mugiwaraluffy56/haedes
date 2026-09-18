package sandbox

import (
	"context"
	"fmt"
	"io"
	"time"
)

const defaultCommandTimeout = 60 * time.Second

type Dependencies struct {
	Repository    SandboxRepository
	Compute       ComputeProvisioner
	Runtime       RuntimeClient
	Snapshots     SnapshotRepository
	SnapshotStore SnapshotStore
	IDs           IDGenerator
	Clock         Clock
}

type Service struct {
	repository    SandboxRepository
	compute       ComputeProvisioner
	runtime       RuntimeClient
	snapshots     SnapshotRepository
	snapshotStore SnapshotStore
	ids           IDGenerator
	clock         Clock
}

func NewService(dependencies Dependencies) *Service {
	return &Service{
		repository:    dependencies.Repository,
		compute:       dependencies.Compute,
		runtime:       dependencies.Runtime,
		snapshots:     dependencies.Snapshots,
		snapshotStore: dependencies.SnapshotStore,
		ids:           dependencies.IDs,
		clock:         dependencies.Clock,
	}
}

func (service *Service) Create(ctx context.Context, ownerID string, config SandboxConfig) (Sandbox, error) {
	if service.repository == nil || service.compute == nil || service.runtime == nil || service.ids == nil || service.clock == nil {
		return Sandbox{}, fmt.Errorf("sandbox service dependencies are incomplete")
	}
	if ownerID == "" || config.MaxLifetime <= 0 {
		return Sandbox{}, fmt.Errorf("owner and max lifetime are required")
	}

	now := service.clock.Now()
	sandbox := Sandbox{
		ID:             service.ids.NewSandboxID(),
		OwnerID:        ownerID,
		State:          StateRequested,
		Config:         config,
		Repository:     config.Repository,
		CreatedAt:      now,
		ExpiresAt:      now.Add(config.MaxLifetime),
		LastActivityAt: now,
		SnapshotIDs:    []SnapshotID{},
	}
	if err := service.repository.Create(ctx, sandbox); err != nil {
		return Sandbox{}, err
	}
	if err := service.transition(ctx, &sandbox, StateProvisioning, ""); err != nil {
		return Sandbox{}, err
	}

	task, err := service.compute.Start(ctx, ProvisionRequest{
		SandboxID: sandbox.ID,
		Config:    config,
		Token:     service.ids.NewRuntimeToken(),
	})
	if err != nil {
		service.fail(ctx, &sandbox, err)
		if task.ARN != "" {
			_ = service.compute.Stop(ctx, task.ARN)
		}
		return Sandbox{}, err
	}
	sandbox.Task = &task
	if err := service.repository.SaveTaskEndpoint(ctx, sandbox.ID, task); err != nil {
		_ = service.compute.Stop(ctx, task.ARN)
		service.fail(ctx, &sandbox, err)
		return Sandbox{}, err
	}
	if err := service.transition(ctx, &sandbox, StateStarting, ""); err != nil {
		_ = service.compute.Stop(ctx, task.ARN)
		return Sandbox{}, err
	}
	if err := service.runtime.WaitReady(ctx, task.Endpoint, task.RuntimeToken); err != nil {
		_ = service.compute.Stop(ctx, task.ARN)
		service.fail(ctx, &sandbox, err)
		return Sandbox{}, err
	}
	if err := service.transition(ctx, &sandbox, StateRunning, ""); err != nil {
		_ = service.compute.Stop(ctx, task.ARN)
		return Sandbox{}, err
	}
	return sandbox, nil
}

func (service *Service) Get(ctx context.Context, ownerID string, id SandboxID) (Sandbox, error) {
	sandbox, err := service.repository.Get(ctx, id)
	if err != nil {
		return Sandbox{}, err
	}
	if sandbox.OwnerID != ownerID {
		return Sandbox{}, ErrNotFound
	}
	return sandbox, nil
}

func (service *Service) List(ctx context.Context, ownerID, cursor string, limit int) (Page[Sandbox], error) {
	return service.repository.List(ctx, ownerID, cursor, limit)
}

func (service *Service) Destroy(ctx context.Context, ownerID string, id SandboxID) (Sandbox, error) {
	sandbox, err := service.Get(ctx, ownerID, id)
	if err != nil {
		return Sandbox{}, err
	}
	if sandbox.State == StateDestroyed {
		return sandbox, nil
	}
	if sandbox.State == StateStopped || sandbox.State == StateFailed {
		if err := service.transition(ctx, &sandbox, StateDestroyed, ""); err != nil {
			return Sandbox{}, err
		}
		return sandbox, nil
	}
	if sandbox.State != StateRunning && sandbox.State != StateStopping {
		return Sandbox{}, InvalidStateTransition{Current: sandbox.State, Requested: StateStopping}
	}
	if sandbox.State == StateRunning {
		if err := service.transition(ctx, &sandbox, StateStopping, "destroy requested"); err != nil {
			return Sandbox{}, err
		}
	}
	if sandbox.Task != nil && sandbox.Task.ARN != "" {
		if err := service.compute.Stop(ctx, sandbox.Task.ARN); err != nil {
			service.fail(ctx, &sandbox, err)
			return Sandbox{}, err
		}
	}
	return sandbox, nil
}

func (service *Service) Execute(ctx context.Context, ownerID string, id SandboxID, request CommandRequest) (CommandResult, error) {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return CommandResult{}, err
	}
	if request.Timeout <= 0 {
		request.Timeout = sandbox.Config.DefaultCommandTimeout
		if request.Timeout <= 0 {
			request.Timeout = defaultCommandTimeout
		}
	}
	return service.runtime.StartCommand(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, request)
}

func (service *Service) SubscribeEvents(ctx context.Context, ownerID string, id SandboxID, commandID CommandID, lastEventID string) (<-chan CommandEvent, error) {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	return service.runtime.Events(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, commandID, lastEventID)
}

func (service *Service) ReadFile(ctx context.Context, ownerID string, id SandboxID, path string) (io.ReadCloser, error) {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	return service.runtime.ReadFile(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, path)
}

func (service *Service) WriteFile(ctx context.Context, ownerID string, id SandboxID, path string, body io.Reader) error {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return err
	}
	return service.runtime.WriteFile(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, path, body)
}

func (service *Service) ListFiles(ctx context.Context, ownerID string, id SandboxID, path string) ([]FileEntry, error) {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	return service.runtime.ListFiles(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, path)
}

func (service *Service) DeleteFile(ctx context.Context, ownerID string, id SandboxID, path string) error {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return err
	}
	return service.runtime.DeleteFile(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, path)
}

func (service *Service) CreateSnapshot(ctx context.Context, ownerID string, id SandboxID, expiresAt *time.Time) (SnapshotMetadata, error) {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return SnapshotMetadata{}, err
	}
	if service.snapshots == nil || service.snapshotStore == nil {
		return SnapshotMetadata{}, fmt.Errorf("snapshot dependencies are incomplete")
	}
	if err := service.transition(ctx, &sandbox, StateSnapshotting, "snapshot requested"); err != nil {
		return SnapshotMetadata{}, err
	}

	now := service.clock.Now()
	snapshot := SnapshotMetadata{
		ID:        service.ids.NewSnapshotID(),
		SandboxID: sandbox.ID,
		State:     "requested",
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := service.snapshots.Create(ctx, snapshot); err != nil {
		service.fail(ctx, &sandbox, err)
		return SnapshotMetadata{}, err
	}
	archive, info, err := service.runtime.ExportWorkspace(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken)
	if err != nil {
		service.fail(ctx, &sandbox, err)
		return SnapshotMetadata{}, err
	}
	stored, err := service.snapshotStore.Put(ctx, snapshot.ID, archive, info)
	_ = archive.Close()
	if err != nil {
		service.fail(ctx, &sandbox, err)
		return SnapshotMetadata{}, err
	}
	snapshot.ObjectKey = string(snapshot.ID)
	snapshot.ByteSize = stored.ByteSize
	snapshot.SHA256 = stored.SHA256
	snapshot.State = "available"
	if err := service.snapshots.UpdateState(ctx, snapshot.ID, "requested", "available"); err != nil {
		service.fail(ctx, &sandbox, err)
		return SnapshotMetadata{}, err
	}
	if err := service.transition(ctx, &sandbox, StateRunning, ""); err != nil {
		return SnapshotMetadata{}, err
	}
	return snapshot, nil
}

func (service *Service) Restore(ctx context.Context, ownerID string, id SandboxID, snapshotID SnapshotID) error {
	sandbox, err := service.runningSandbox(ctx, ownerID, id)
	if err != nil {
		return err
	}
	if service.snapshots == nil || service.snapshotStore == nil {
		return fmt.Errorf("snapshot dependencies are incomplete")
	}
	snapshot, err := service.snapshots.Get(ctx, snapshotID)
	if err != nil {
		return err
	}
	if snapshot.State != "available" {
		return fmt.Errorf("snapshot is not available")
	}
	ownerSandbox, err := service.Get(ctx, ownerID, snapshot.SandboxID)
	if err != nil || ownerSandbox.OwnerID != sandbox.OwnerID {
		return ErrNotFound
	}
	archive, _, err := service.snapshotStore.Open(ctx, snapshotID)
	if err != nil {
		return err
	}
	defer archive.Close()
	return service.runtime.RestoreWorkspace(ctx, sandbox.Task.Endpoint, sandbox.Task.RuntimeToken, archive)
}

func (service *Service) runningSandbox(ctx context.Context, ownerID string, id SandboxID) (Sandbox, error) {
	sandbox, err := service.Get(ctx, ownerID, id)
	if err != nil {
		return Sandbox{}, err
	}
	if err := RequireRunning(sandbox); err != nil {
		return Sandbox{}, err
	}
	if sandbox.Task == nil || sandbox.Task.Endpoint == "" || sandbox.Task.RuntimeToken == "" {
		return Sandbox{}, ErrRuntimeMissing
	}
	return sandbox, nil
}

func (service *Service) transition(ctx context.Context, sandbox *Sandbox, next State, reason string) error {
	candidate := *sandbox
	if err := Transition(&candidate, next, reason); err != nil {
		return err
	}
	if err := service.repository.UpdateState(ctx, sandbox.ID, sandbox.State, next, reason); err != nil {
		return err
	}
	*sandbox = candidate
	return nil
}

func (service *Service) fail(ctx context.Context, sandbox *Sandbox, cause error) {
	if CanTransition(sandbox.State, StateFailed) {
		_ = service.transition(ctx, sandbox, StateFailed, cause.Error())
	}
}
