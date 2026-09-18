package sandbox

import (
	"io"
	"time"
)

type State string

const (
	StateRequested    State = "requested"
	StateProvisioning State = "provisioning"
	StateStarting     State = "starting"
	StateRunning      State = "running"
	StateSnapshotting State = "snapshotting"
	StateStopping     State = "stopping"
	StateStopped      State = "stopped"
	StateFailed       State = "failed"
	StateDestroyed    State = "destroyed"
)

type SandboxID string
type CommandID string
type SnapshotID string

type Page[T any] struct {
	Items      []T
	NextCursor string
}

type Sandbox struct {
	ID             SandboxID
	OwnerID        string
	State          State
	Config         SandboxConfig
	Repository     *RepositoryConfig
	Task           *TaskRef
	CreatedAt      time.Time
	ExpiresAt      time.Time
	LastActivityAt time.Time
	CurrentCommand *CommandID
	SnapshotIDs    []SnapshotID
	FailureReason  string
}

type SandboxConfig struct {
	Image                 string
	CPUMillis             int
	MemoryMiB             int
	StorageGiB            int
	MaxLifetime           time.Duration
	DefaultCommandTimeout time.Duration
	Environment           map[string]string
}

type RepositoryConfig struct {
	Provider string
	URL      string
	Ref      string
	Path     string
}

type TaskRef struct {
	ARN          string
	PrivateIP    string
	Endpoint     string
	RuntimeToken string
}

type ProvisionRequest struct {
	SandboxID SandboxID
	Config    SandboxConfig
	Token     string
}

type CommandRequest struct {
	Command     string
	CWD         string
	Environment map[string]string
	Timeout     time.Duration
}

type CommandResult struct {
	ID         CommandID
	SandboxID  SandboxID
	Command    string
	ExitCode   *int
	StartedAt  time.Time
	FinishedAt *time.Time
	TimedOut   bool
	Signal     string
}

type CommandEvent struct {
	Sequence  int64
	Type      string
	CommandID CommandID
	Data      string
	Result    *CommandResult
	At        time.Time
}

type FileEntry struct {
	Path       string
	Kind       string
	ByteSize   int64
	ModifiedAt time.Time
}

type SnapshotMetadata struct {
	ID        SnapshotID
	SandboxID SandboxID
	State     string
	ObjectKey string
	ByteSize  int64
	SHA256    string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type TaskStatus struct {
	ARN       string
	Phase     string
	PrivateIP string
	Reason    string
}

type CommandHandle struct {
	ID CommandID
}

type ArchiveInfo struct {
	ByteSize  int64
	SHA256    string
	MediaType string
}

type Principal struct {
	OwnerID string
	KeyID   string
}

type RuntimeArchive struct {
	Content io.ReadCloser
	Info    ArchiveInfo
}
