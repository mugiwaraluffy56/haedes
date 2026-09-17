// Code generated from internal/contracts/api.openapi.yaml; DO NOT EDIT.

package contracts

import "time"

type ApiError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID RequestID      `json:"requestId"`
	Details   map[string]any `json:"details"`
}

type CommandCompletedEvent struct {
	Type      string        `json:"type"`
	CommandID CommandID     `json:"commandId"`
	Result    CommandResult `json:"result"`
	At        Timestamp     `json:"at"`
}

type CommandEvent interface {
	isCommandEvent()
}

type CommandFailedEvent struct {
	Type      string    `json:"type"`
	CommandID CommandID `json:"commandId"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	At        Timestamp `json:"at"`
}

type CommandID = string

type CommandOutputEvent struct {
	Type      string    `json:"type"`
	CommandID CommandID `json:"commandId"`
	Data      string    `json:"data"`
	At        Timestamp `json:"at"`
}

type CommandRequest struct {
	Command        string             `json:"command"`
	Cwd            *WorkspacePath     `json:"cwd,omitempty"`
	Environment    *map[string]string `json:"environment,omitempty"`
	TimeoutSeconds *int32             `json:"timeoutSeconds,omitempty"`
}

type CommandResult struct {
	ID         CommandID  `json:"id"`
	SandboxID  SandboxID  `json:"sandboxId"`
	Command    string     `json:"command"`
	ExitCode   *int32     `json:"exitCode"`
	StartedAt  Timestamp  `json:"startedAt"`
	FinishedAt *Timestamp `json:"finishedAt,omitempty"`
	TimedOut   bool       `json:"timedOut"`
	Signal     *string    `json:"signal,omitempty"`
}

type CommandStartedEvent struct {
	Type      string    `json:"type"`
	CommandID CommandID `json:"commandId"`
	At        Timestamp `json:"at"`
}

type ErrorResponse struct {
	Error ApiError `json:"error"`
}

type FileEntry struct {
	Path       WorkspaceFilePath `json:"path"`
	Kind       string            `json:"kind"`
	ByteSize   *int32            `json:"byteSize,omitempty"`
	ModifiedAt *Timestamp        `json:"modifiedAt,omitempty"`
}

type FileListResponse struct {
	Path    WorkspacePath `json:"path"`
	Entries []FileEntry   `json:"entries"`
}

type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type PageInfo struct {
	NextCursor *string `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

type RepositoryConfig struct {
	Provider string        `json:"provider"`
	URL      string        `json:"url"`
	Ref      *string       `json:"ref,omitempty"`
	Path     WorkspacePath `json:"path"`
}

type RequestID = string

type RestoreRequest struct {
	SnapshotID SnapshotID `json:"snapshotId"`
}

type Sandbox struct {
	ID               SandboxID         `json:"id"`
	State            SandboxState      `json:"state"`
	Repository       *RepositoryConfig `json:"repository,omitempty"`
	Config           SandboxConfig     `json:"config"`
	CreatedAt        Timestamp         `json:"createdAt"`
	ExpiresAt        Timestamp         `json:"expiresAt"`
	LastActivityAt   Timestamp         `json:"lastActivityAt"`
	CurrentCommandID *CommandID        `json:"currentCommandId,omitempty"`
	SnapshotIds      []SnapshotID      `json:"snapshotIds"`
}

type SandboxConfig struct {
	Image                        string            `json:"image"`
	CpuMillis                    int32             `json:"cpuMillis"`
	MemoryMiB                    int32             `json:"memoryMiB"`
	StorageGiB                   int32             `json:"storageGiB"`
	MaxLifetimeSeconds           int32             `json:"maxLifetimeSeconds"`
	DefaultCommandTimeoutSeconds int32             `json:"defaultCommandTimeoutSeconds"`
	Environment                  map[string]string `json:"environment"`
	Repository                   *RepositoryConfig `json:"repository,omitempty"`
	SnapshotID                   *SnapshotID       `json:"snapshotId,omitempty"`
}

type SandboxCreateRequest struct {
	Config SandboxConfig `json:"config"`
}

type SandboxID = string

type SandboxPage struct {
	Items []Sandbox `json:"items"`
	Page  PageInfo  `json:"page"`
}

type SandboxState string

const SandboxStateRequested SandboxState = "requested"
const SandboxStateProvisioning SandboxState = "provisioning"
const SandboxStateStarting SandboxState = "starting"
const SandboxStateRunning SandboxState = "running"
const SandboxStateSnapshotting SandboxState = "snapshotting"
const SandboxStateStopping SandboxState = "stopping"
const SandboxStateStopped SandboxState = "stopped"
const SandboxStateFailed SandboxState = "failed"
const SandboxStateDestroyed SandboxState = "destroyed"

type SnapshotCreateRequest struct {
	ExpiresAt *Timestamp `json:"expiresAt,omitempty"`
}

type SnapshotID = string

type SnapshotMetadata struct {
	ID        SnapshotID `json:"id"`
	SandboxID SandboxID  `json:"sandboxId"`
	State     string     `json:"state"`
	// Opaque archive reference; not an S3 URL.
	ObjectKey string     `json:"objectKey"`
	ByteSize  *int32     `json:"byteSize,omitempty"`
	Checksum  *string    `json:"checksum,omitempty"`
	CreatedAt Timestamp  `json:"createdAt"`
	ExpiresAt *Timestamp `json:"expiresAt,omitempty"`
}

type SnapshotPage struct {
	Items []SnapshotMetadata `json:"items"`
	Page  PageInfo           `json:"page"`
}

type Timestamp = time.Time

type WorkspaceFilePath = string

type WorkspacePath = string

func (CommandStartedEvent) isCommandEvent()   {}
func (CommandOutputEvent) isCommandEvent()    {}
func (CommandCompletedEvent) isCommandEvent() {}
func (CommandFailedEvent) isCommandEvent()    {}
