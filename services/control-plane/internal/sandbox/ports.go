package sandbox

import (
	"context"
	"io"
	"time"
)

type SandboxRepository interface {
	Create(ctx context.Context, sandbox Sandbox) error
	Get(ctx context.Context, id SandboxID) (Sandbox, error)
	List(ctx context.Context, ownerID, cursor string, limit int) (Page[Sandbox], error)
	UpdateState(ctx context.Context, id SandboxID, expected, next State, reason string) error
	SaveTaskEndpoint(ctx context.Context, id SandboxID, task TaskRef) error
}

type ComputeProvisioner interface {
	Start(ctx context.Context, request ProvisionRequest) (TaskRef, error)
	Stop(ctx context.Context, taskARN string) error
	Describe(ctx context.Context, taskARN string) (TaskStatus, error)
}

type RuntimeClient interface {
	WaitReady(ctx context.Context, endpoint, token string) error
	StartCommand(ctx context.Context, endpoint, token string, request CommandRequest) (CommandResult, error)
	Events(ctx context.Context, endpoint, token string, commandID CommandID, lastEventID string) (<-chan CommandEvent, error)
	ReadFile(ctx context.Context, endpoint, token, path string) (io.ReadCloser, error)
	WriteFile(ctx context.Context, endpoint, token, path string, body io.Reader) error
	ListFiles(ctx context.Context, endpoint, token, path string) ([]FileEntry, error)
	DeleteFile(ctx context.Context, endpoint, token, path string) error
	ExportWorkspace(ctx context.Context, endpoint, token string) (io.ReadCloser, ArchiveInfo, error)
	RestoreWorkspace(ctx context.Context, endpoint, token string, archive io.Reader) error
}

type SnapshotRepository interface {
	Create(ctx context.Context, snapshot SnapshotMetadata) error
	Get(ctx context.Context, id SnapshotID) (SnapshotMetadata, error)
	UpdateState(ctx context.Context, id SnapshotID, expected, next string) error
}

type SnapshotStore interface {
	Put(ctx context.Context, id SnapshotID, archive io.Reader, info ArchiveInfo) (ArchiveInfo, error)
	Open(ctx context.Context, id SnapshotID) (io.ReadCloser, ArchiveInfo, error)
}

type IDGenerator interface {
	NewSandboxID() SandboxID
	NewCommandID() CommandID
	NewSnapshotID() SnapshotID
	NewRuntimeToken() string
}

type Clock interface {
	Now() time.Time
}

type AuthService interface {
	Authenticate(ctx context.Context, apiKey string) (Principal, error)
}
