# haedes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-shaped monorepo and a demonstrable vertical slice in which one real coding agent requests a temporary AWS computer through MCP, executes and streams real commands, changes a repository, saves the workspace to S3, destroys the Fargate task, restores it into a new task, and continues working. The TypeScript SDK and dashboard remain important integration and control surfaces underneath that agent-native experience.

**Architecture:** Agents reach a thin MCP adapter, which delegates to the TypeScript SDK or a shared HTTP client. The versioned HTTP API is the canonical universal platform contract for MCP, the SDK, dashboard, future adapters, customer integrations, and internal tools. The Go control plane owns sandbox identity, lifecycle state, authorization, AWS orchestration, metadata, event fan-out, and snapshot coordination. Each Fargate task contains only a Rust runtime that owns `/workspace`, child processes, filesystem operations, command streaming, and archive import/export. Terraform creates the AWS resources that connect the control plane, sandbox tasks, S3, DynamoDB, ECR, CloudWatch, and VPC.

**Tech Stack:** TypeScript, Next.js, React, pnpm workspaces, Turborepo, Go, chi, AWS SDK for Go v2, Rust, Tokio, Axum, serde, Docker, Amazon ECS with AWS Fargate, Amazon ECR, Amazon S3, Amazon DynamoDB, Amazon CloudWatch, Amazon VPC, Terraform, GitHub Actions, Vitest, Playwright, Go test, Rust test, and k6.

**Spec:** `docs/prd.md`, with hackathon constraints from `docs/hackathon.md` and planning rules from `docs/skill.md`.

## Global Constraints

- The product abstraction is a disposable Linux computer for an AI agent, not a Docker, ECS, Fargate, or AWS SDK abstraction.
- The agent is the brain; haedes is the computer it works on. The dashboard is a human observability and control surface, not the primary product entry point.
- The integration hierarchy is agent-native MCP and adapters, TypeScript SDK, and canonical versioned HTTP API. The hackathon implements one MCP path and one real coding-agent client; it does not build multiple agent plugins or a generic agent framework.
- AWS must be part of the product behavior: sandboxes run on ECS/Fargate, snapshots use S3, metadata uses DynamoDB, logs use CloudWatch, images use ECR, and networking uses VPC.
- The first working path must cover sandbox creation, destruction, command execution, live output, filesystem operations, Git cloning, dependency installation, resource timeout, S3 snapshotting, restoration, a dashboard, execution history, a TypeScript SDK, and one real coding-agent integration.
- Local Docker is for developing and testing the product only; there is no user-facing local sandbox mode.
- The initial sandbox image supports Git, curl, wget, bash, Python, pip, Node.js, npm, Go, Rust, gcc, g++, make, and common Linux utilities.
- Sandbox lifecycle states are `requested`, `provisioning`, `starting`, `running`, `snapshotting`, `stopping`, `stopped`, `failed`, and `destroyed`.
- Every sandbox has an expiration time, every command has a timeout, and both limits are enforced server-side.
- Runtime code never contains user, billing, dashboard, or application-level logic.
- AWS SDK calls stay behind Go interfaces and adapters; HTTP handlers never call AWS directly.
- `/workspace` is the only user filesystem root. File APIs reject traversal, symlink escapes, and paths outside that root.
- Sandbox tasks receive no AWS credentials. The control plane owns S3, DynamoDB, and ECS permissions.
- Runtime access uses a short-lived, sandbox-scoped internal token. Public API keys are never forwarded to a runtime task.
- The MCP server holds only a server-side API credential for the public control plane. Tool inputs cannot provide AWS credentials or runtime tokens, and tool outputs redact credentials and implementation-only AWS identifiers.
- MCP authenticates and authorizes by using the same public HTTP API as every other client; it does not maintain a parallel identity, lifecycle, or persistence boundary.
- The first demo uses one default development image, one AWS region, one account, and one coding-agent workflow.
- The real coding-agent client is selected only after a compatibility spike verifies that it can consume the chosen MCP integration in the actual demo environment. Do not claim Claude Code, Codex, or another vendor integration is complete without that verification.
- GPU environments, browser automation, Kubernetes, custom image management, billing, enterprise authentication, organizations, multiple regions, marketplace features, advanced networking controls, and autoscaling are outside the first release.
- Every task ends with an independently runnable test command and a focused commit.

## Product decisions to lock before coding

1. Use one monorepo with pnpm and Turborepo for JavaScript projects, a root `go.work` for Go services, and a Cargo workspace for Rust projects. Each service still has its own package/module boundary. The MCP adapter lives at `integrations/mcp` because it is an integration owned by the agent-facing TypeScript boundary, not a control-plane service.
2. Use OpenAPI 3.1 as the canonical public API contract at `internal/contracts/api.openapi.yaml`. Generate TypeScript types from it and generate Go server models from it. Runtime messages use a separate versioned JSON schema because the runtime is an internal protocol, not a public API.
3. Use HTTP JSON for request/response operations and Server-Sent Events for command output and lifecycle events. SSE keeps the browser and SDK simple while preserving ordered events and reconnect support.
4. Treat `POST /v1/sandboxes` and `POST /v1/sandboxes/{id}/snapshots` as idempotent when an `Idempotency-Key` is supplied. Repeating a request returns the original resource instead of creating a second sandbox or snapshot.
5. Model a sandbox as one ECS task with one runtime container. The control plane stores the private task endpoint after ECS reports the task network attachment.
6. Store snapshot archives as compressed tar streams in S3. The archive root is `/workspace`; restore rejects absolute paths, `..` components, and symlinks that resolve outside the workspace.
7. Start with one public ALB for the control plane and private subnets for control-plane and sandbox tasks. Sandbox security groups allow runtime traffic only from the control-plane security group. NAT provides outbound package and Git access in the development environment.
8. Use an in-memory repository and fake AWS/runtime adapters for fast local tests. The production adapter selection is explicit through configuration and cannot silently fall back to mocks.
9. Keep MCP thin: for the hackathon it delegates to `@haedes/sdk` so it reuses the product abstraction and HTTP/SSE behavior. If MCP transport constraints require a direct HTTP client, share the client and generated API types rather than duplicating domain models or lifecycle logic.

## Repository shape

The following directories have a concrete responsibility. Do not create a directory unless it owns code, contracts, infrastructure, examples, or tests described here.

```text
haedes/
├── apps/
│   ├── web/                         # Public landing page and product explanation
│   └── dashboard/                   # Sandbox management UI and live activity view
├── services/
│   ├── control-plane/               # Public Go API and orchestration
│   ├── sandbox-runtime/             # Rust process/filesystem service inside a task
│   ├── lifecycle-manager/           # Go expiration and reconciliation worker
│   └── snapshot-service/            # Go workspace archive and S3 coordination boundary
├── packages/
│   ├── sdk-typescript/              # Developer-facing sandbox abstraction
│   ├── api-types/                   # Generated public API types and validation helpers
│   ├── shared-config/               # Shared TypeScript lint, format, and build config
│   └── protocol/                    # Versioned runtime protocol schema and fixtures
├── infrastructure/
│   ├── terraform/                   # Reusable AWS modules and dev/prod compositions
│   ├── aws/                         # AWS runbooks, IAM notes, and console verification
│   ├── docker/                      # Local service composition and development images
│   └── environments/                # Environment variable and deployment contracts
├── images/
│   ├── sandbox-base/                # Reproducible user-facing Linux developer image
│   └── sandbox-dev/                 # Image used by local runtime/integration tests
├── integrations/
│   ├── github/                      # Repository URL and credential boundary
│   ├── mcp/                         # Initial agent-native MCP adapter over the SDK/API
│   ├── agents/                      # Provider-neutral future adapter contracts only
│   └── examples/                    # Runnable provider-neutral agent examples
├── internal/
│   ├── contracts/                   # Canonical OpenAPI and lifecycle contract sources
│   ├── schemas/                     # JSON schemas for persisted and wire payloads
│   └── fixtures/                    # Small repositories, commands, and API fixtures
├── scripts/                         # Repeatable setup, generation, validation, and demo scripts
├── tests/
│   ├── integration/                 # Cross-service tests using local fakes or containers
│   ├── e2e/                         # Browser and SDK user journeys
│   ├── security/                    # Boundary and abuse-case tests
│   └── load/                        # Command and lifecycle load scenarios
├── docs/
│   ├── architecture/                # System, data-flow, and deployment diagrams
│   ├── api/                         # Public API and SDK usage documentation
│   ├── sandbox/                     # Runtime behavior and image documentation
│   ├── aws/                         # AWS deployment and demo verification runbooks
│   ├── security/                    # Threat model and security operating rules
│   └── decisions/                   # Short architecture decision records
├── examples/
│   ├── basic-agent/                 # Create, exec, inspect, and destroy
│   ├── coding-agent/                # Clone, test, edit, snapshot, restore, retest
│   └── github-fixer/                # Issue-to-workspace integration example
├── .github/workflows/               # CI and infrastructure validation
├── package.json
├── pnpm-workspace.yaml
├── turbo.json
├── go.work
├── Cargo.toml
├── Makefile
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── LICENSE
└── .env.example
```

`integrations/mcp` owns MCP transport, tool schemas, input validation, structured tool results, platform-error mapping, and adapter tests. It does not own sandbox state, AWS orchestration, runtime access, persistence, or a model-provider loop. `services/control-plane` remains the lifecycle owner. `packages/sdk-typescript` remains the developer-facing product abstraction. The `integrations/agents` directory may document provider-neutral host contracts, but the hackathon must build only one MCP implementation.

## Public concepts and contracts

### Core types

The canonical names below must be reused by Go, TypeScript, Rust protocol code, dashboard state, fixtures, and documentation:

```typescript
type SandboxState =
  | "requested"
  | "provisioning"
  | "starting"
  | "running"
  | "snapshotting"
  | "stopping"
  | "stopped"
  | "failed"
  | "destroyed";

interface SandboxConfig {
  image: string;
  cpuMillis: number;
  memoryMiB: number;
  storageGiB: number;
  maxLifetimeSeconds: number;
  defaultCommandTimeoutSeconds: number;
  environment: Record<string, string>;
  repository?: RepositoryConfig;
  snapshotId?: string;
}

interface RepositoryConfig {
  provider: "github";
  url: string;
  ref?: string;
  path: string;
}

interface Sandbox {
  id: string;
  state: SandboxState;
  repository?: RepositoryConfig;
  config: SandboxConfig;
  createdAt: string;
  expiresAt: string;
  lastActivityAt: string;
  currentCommandId?: string;
  snapshotIds: string[];
}

interface CommandRequest {
  command: string;
  cwd?: string;
  environment?: Record<string, string>;
  timeoutSeconds?: number;
}

interface CommandResult {
  id: string;
  sandboxId: string;
  command: string;
  exitCode: number | null;
  startedAt: string;
  finishedAt?: string;
  timedOut: boolean;
  signal?: string;
}

type CommandEvent =
  | { type: "started"; commandId: string; at: string }
  | { type: "stdout"; commandId: string; data: string; at: string }
  | { type: "stderr"; commandId: string; data: string; at: string }
  | { type: "completed"; commandId: string; result: CommandResult; at: string }
  | { type: "failed"; commandId: string; code: string; message: string; at: string };

interface SnapshotMetadata {
  id: string;
  sandboxId: string;
  state: "requested" | "creating" | "available" | "failed" | "deleted";
  objectKey: string;
  byteSize?: number;
  checksum?: string;
  createdAt: string;
  expiresAt?: string;
}

interface FileEntry {
  path: string;
  kind: "file" | "directory" | "symlink";
  byteSize?: number;
  modifiedAt?: string;
}

interface ApiError {
  error: {
    code: string;
    message: string;
    requestId: string;
    details: Record<string, unknown>;
  };
}
```

### Lifecycle transition table

| Current state | Allowed next state | Trigger | Required side effect |
| --- | --- | --- | --- |
| `requested` | `provisioning`, `failed` | create accepted or validation/provisioning failure | Persist resource before starting ECS work |
| `provisioning` | `starting`, `failed` | ECS task has a task ARN and endpoint or launch fails | Persist task ARN and private endpoint when available |
| `starting` | `running`, `failed` | runtime health check succeeds or startup deadline expires | Record `lastActivityAt` when healthy |
| `running` | `snapshotting`, `stopping`, `failed` | snapshot, destroy/expiry, or unrecoverable runtime error | Reject commands once transition begins |
| `snapshotting` | `running`, `failed` | archive stored or snapshot fails | Keep prior workspace intact on failed upload |
| `stopping` | `stopped`, `destroyed`, `failed` | ECS task stops or stop deadline expires | Reconcile actual ECS status before final state |
| `stopped` | `destroyed` | explicit destroy or retention cleanup | Ensure the task is absent before marking destroyed |
| `failed` | `destroyed` | cleanup request | Best-effort cleanup, retain failure reason |
| `destroyed` | none | terminal state | Return conflict for commands, snapshots, or restore-in-place |

The state machine must reject every unlisted transition and must make retries safe. A repeated destroy request is successful if the resource is already `destroyed`; a repeated create request is safe only when the same idempotency key is present.

### Public API surface

The OpenAPI contract must define these routes and error envelopes:

```text
POST   /v1/sandboxes
GET    /v1/sandboxes
GET    /v1/sandboxes/{sandboxId}
DELETE /v1/sandboxes/{sandboxId}

POST   /v1/sandboxes/{sandboxId}/commands
GET    /v1/sandboxes/{sandboxId}/commands/{commandId}
GET    /v1/sandboxes/{sandboxId}/commands/{commandId}/events

GET    /v1/sandboxes/{sandboxId}/files?path=/workspace
GET    /v1/sandboxes/{sandboxId}/files/content?path=/workspace/file
PUT    /v1/sandboxes/{sandboxId}/files/content?path=/workspace/file
DELETE /v1/sandboxes/{sandboxId}/files/content?path=/workspace/file

POST   /v1/sandboxes/{sandboxId}/snapshots
GET    /v1/sandboxes/{sandboxId}/snapshots
GET    /v1/snapshots/{snapshotId}
POST   /v1/sandboxes/{sandboxId}/restore

GET    /healthz
GET    /readyz
```

Every non-2xx response uses:

```json
{
  "error": {
    "code": "sandbox_not_running",
    "message": "Sandbox sbx_123 is provisioning",
    "requestId": "req_123",
    "details": {}
  }
}
```

`POST /commands` returns `202 Accepted` with a `CommandResult` containing a command ID. `GET /events` emits `text/event-stream`; each event has `id`, `event`, and JSON `data`. The terminal completion event is replayable from `Last-Event-ID` while the command record is retained.

### Runtime protocol

The internal protocol is versioned under `/v1` and is never exposed by the public load balancer:

```text
GET    /healthz
POST   /v1/exec
GET    /v1/exec/{commandId}/events
GET    /v1/files?path=/workspace
GET    /v1/files/content?path=/workspace/file
PUT    /v1/files/content?path=/workspace/file
DELETE /v1/files/content?path=/workspace/file
POST   /v1/snapshot/export
POST   /v1/snapshot/restore
```

The control plane translates public API types into runtime protocol types. The runtime never receives a public API key, user identity, or dashboard request.

### Agent-native MCP contract

The hackathon MCP adapter exposes one tool for each product operation, using the public sandbox vocabulary rather than AWS vocabulary:

```text
sandbox_create
sandbox_get
sandbox_list
sandbox_exec
sandbox_exec_stream
sandbox_read_file
sandbox_write_file
sandbox_list_files
sandbox_delete_file
sandbox_snapshot
sandbox_restore
sandbox_destroy
```

Each tool has a versioned input and output schema owned by `integrations/mcp`. Inputs reuse the canonical public types wherever possible; the adapter adds only MCP-specific descriptions and result envelopes. Tool results include the sandbox ID and, for command operations, the command ID and structured status. Streaming exposes ordered stdout/stderr and terminal events through the MCP mechanism supported by the selected client, while the HTTP API remains the source of truth for replay and reconnection semantics.

The adapter authenticates to the public API with a server-side configured API key. That key is never accepted as tool input, returned to the agent, or forwarded to a sandbox task. MCP maps `ApiError` codes into stable tool errors, including authentication failures, nonexistent sandboxes, state conflicts, command timeouts, and runtime failures. It never calls ECS, S3, DynamoDB, CloudWatch, or the Rust runtime directly and never persists lifecycle state independently.

## Task plan

### Task 1: Establish repository tooling and ownership boundaries

**Files:**

- Create: `package.json`, `pnpm-workspace.yaml`, `turbo.json`
- Create: `go.work`
- Create: `Cargo.toml`
- Create: `Makefile`, `.env.example`, `.gitignore`
- Create: `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE`, `README.md`
- Create: `scripts/check-repo.sh`, `scripts/generate-contracts.sh`
- Create: `docs/architecture/repository.md`, `docs/decisions/0001-monorepo-boundaries.md`

**Interfaces:**

- Produces `pnpm lint`, `pnpm test`, `pnpm typecheck`, `go test ./...`, `cargo test --workspace`, `make check`, and `make generate` entry points.
- Produces a workspace where JavaScript packages are discovered by pnpm, Go services are listed in `go.work`, and Rust crates are members of the Cargo workspace.

- [ ] **Step 1: Write the root package and workspace manifests.** Add only `apps/*`, `packages/*`, `integrations/mcp`, and `integrations/examples/*` as pnpm workspace packages. Keep the GitHub and provider-neutral adapter directories as owned source boundaries unless they need a package manifest. Pin the package manager version and Node version in `package.json` and `.nvmrc`.
- [ ] **Step 2: Add the Go and Rust workspace members.** List each service `go.mod` in `go.work` and each Rust service in the root Cargo workspace without adding service implementation files yet.
- [ ] **Step 3: Define root validation commands.** Make `make check` run manifest checks, TypeScript checks, Go tests, and Rust tests; make `make generate` run contract generation and fail when generated files differ from source.
- [ ] **Step 4: Document ownership.** State that public UI and SDK changes belong to the TypeScript owners, MCP tool schemas and adapter behavior belong to the `integrations/mcp` owners, orchestration changes belong to the control-plane owners, runtime changes belong to the Rust owners, and AWS changes require Terraform plus runbook updates. MCP changes must be reviewed against the public API contract and must not add AWS or lifecycle logic.
- [ ] **Step 5: Test the foundation.** Run `make check` and verify it reports missing service manifests as expected until Tasks 2 through 4 add them, then commit the manifests and documentation as `chore: establish monorepo foundations`.

### Task 2: Define canonical contracts and lifecycle rules

**Files:**

- Create: `internal/contracts/api.openapi.yaml`
- Create: `internal/contracts/lifecycle.md`
- Create: `internal/schemas/sandbox.schema.json`, `internal/schemas/command.schema.json`, `internal/schemas/snapshot.schema.json`
- Create: `packages/api-types/package.json`, `packages/api-types/src/index.ts`, `packages/api-types/src/generated.ts`
- Create: `packages/protocol/package.json`, `packages/protocol/src/runtime-v1.schema.json`, `packages/protocol/src/fixtures/*.json`
- Create: `scripts/validate-contracts.ts`
- Create: `integrations/mcp/src/tool-contract.schema.json`
- Create: `docs/api/public-api.md`, `docs/api/runtime-protocol.md`, `docs/architecture/state-machine.md`

**Interfaces:**

- Produces `SandboxState`, `SandboxConfig`, `CommandRequest`, `CommandResult`, `CommandEvent`, `SnapshotMetadata`, and `ApiError` with the exact fields listed above.
- Produces generated TypeScript exports from `packages/api-types/src/index.ts`; generated output is not edited by hand.
- Produces runtime schema version `runtime.v1` and fixtures for command start, stdout, stderr, completion, file operations, export, and restore.
- Produces a small MCP tool contract that validates tool names, required inputs, structured outputs, sandbox/command identity fields, and mapped platform error fields without creating a second domain model.

- [ ] **Step 1: Write failing contract validation tests.** Test that every lifecycle state is present, every legal transition is accepted, every illegal transition is rejected, and `ApiError` requires `code`, `message`, and `requestId`.
- [ ] **Step 2: Write the OpenAPI contract.** Define request/response schemas, pagination fields, idempotency headers, SSE event payloads, `409` state conflicts, `401` authentication failures, `404` missing resources, `408` command timeout results, and `413` payload limits.
- [ ] **Step 3: Add JSON schemas and runtime fixtures.** Require bounded command length, bounded environment size, bounded file body size, absolute timestamps in RFC 3339 format, and opaque IDs with prefixes `sbx_`, `cmd_`, and `snp_`.
- [ ] **Step 4: Generate TypeScript types and validation helpers.** Run `pnpm exec openapi-typescript internal/contracts/api.openapi.yaml -o packages/api-types/src/generated.ts`; export schemas and a `parseApiError` helper from `packages/api-types/src/index.ts`.
- [ ] **Step 5: Define and validate the MCP contract.** Add tool schemas and structured result examples for create, exec, stream, file operations, snapshot, restore, and destroy. Validate them against the generated public types and reject undocumented tool fields.
- [ ] **Step 6: Run contract tests.** Run `pnpm validate:contracts` and `pnpm typecheck`; verify the generated file is reproducible and commit as `feat: define public and runtime contracts`.

### Task 3: Build the Rust sandbox runtime core

**Files:**

- Create: `services/sandbox-runtime/Cargo.toml`
- Create: `services/sandbox-runtime/src/main.rs`
- Create: `services/sandbox-runtime/src/config.rs`, `src/health.rs`, `src/server/mod.rs`
- Create: `services/sandbox-runtime/src/filesystem/mod.rs`, `src/filesystem/paths.rs`, `src/filesystem/service.rs`
- Create: `services/sandbox-runtime/src/process/mod.rs`, `src/process/runner.rs`
- Create: `services/sandbox-runtime/src/execution/mod.rs`, `src/execution/service.rs`
- Create: `services/sandbox-runtime/src/streaming/mod.rs`, `src/streaming/bus.rs`
- Create: `services/sandbox-runtime/src/snapshot/mod.rs`, `src/snapshot/archive.rs`
- Create: `services/sandbox-runtime/tests/runtime_http.rs`, `tests/path_security.rs`, `tests/timeout.rs`
- Create: `services/sandbox-runtime/README.md`

**Interfaces:**

- Produces these runtime error codes: `path_outside_workspace`, `file_not_found`, `body_too_large`, `command_timeout`, `command_output_limit`, `runtime_unauthorized`, `archive_entry_unsafe`, and `runtime_unavailable`.

```rust
pub trait PathGuard {
    fn resolve(&self, user_path: &str) -> Result<PathBuf, RuntimeError>;
}

pub struct CommandRequest {
    pub command: String,
    pub cwd: Option<String>,
    pub environment: HashMap<String, String>,
    pub timeout: Duration,
}

pub struct CommandHandle {
    pub id: String,
}

pub enum RuntimeError {
    PathOutsideWorkspace,
    FileNotFound,
    BodyTooLarge,
    CommandTimeout,
    CommandOutputLimit,
    RuntimeUnauthorized,
    ArchiveEntryUnsafe,
    RuntimeUnavailable,
}
```

- Produces `PathGuard::resolve(user_path) -> Result<PathBuf, RuntimeError>` rooted at `/workspace`.
- Produces `CommandRunner::start(CommandRequest) -> Result<CommandHandle, RuntimeError>` and `CommandRunner::events(command_id) -> Receiver<CommandEvent>`.
- Produces Axum handlers for `/healthz`, `/v1/exec`, `/v1/exec/:command_id/events`, `/v1/files`, `/v1/files/content`, `/v1/snapshot/export`, and `/v1/snapshot/restore`.

- [ ] **Step 1: Write failing path tests.** Cover `/workspace/a.txt`, `a.txt`, `/workspace/../etc/passwd`, encoded traversal, symlink-to-outside, and a missing parent directory. Expected: only paths within `/workspace` resolve.
- [ ] **Step 2: Implement `PathGuard`.** Canonicalize existing ancestors, reject NUL bytes and traversal, create parent directories only beneath `/workspace`, and return typed `path_outside_workspace` errors.
- [ ] **Step 3: Write failing filesystem tests.** Test read, write, list, delete, empty files, UTF-8 content, missing files, and the configured body-size limit.
- [ ] **Step 4: Implement filesystem handlers.** Use Tokio file operations, return directory entries with type and byte size, and never follow a symlink outside the root.
- [ ] **Step 5: Write failing command tests.** Test stdout and stderr ordering within each stream, exit code `0`, non-zero exit code, working directory, environment injection, shell metacharacters as command input, and timeout termination.
- [ ] **Step 6: Implement process execution.** Run commands through `sh -lc` inside the task, place each command in its own process group, pipe stdout/stderr, cap output bytes, terminate the group on timeout, and publish `started`, stream, and terminal `completed` events.
- [ ] **Step 7: Add authenticated HTTP handlers.** Require the runtime token on every `/v1/*` route, permit unauthenticated `/healthz`, return the exact contract error envelope, and attach a monotonic event sequence for SSE replay.
- [ ] **Step 8: Implement archive export and restore.** Export `/workspace` as tar plus zstd, exclude only configured cache directories, preserve regular files and directories, reject unsafe archive entries during restore, and calculate a SHA-256 checksum.
- [ ] **Step 9: Run runtime tests.** Run `cargo fmt --check`, `cargo clippy --all-targets --all-features -- -D warnings`, and `cargo test`; commit as `feat: add sandbox runtime execution core`.

### Task 4: Create the Go control-plane domain and ports

**Files:**

- Create: `services/control-plane/go.mod`
- Create: `services/control-plane/cmd/server/main.go`
- Create: `services/control-plane/internal/config/config.go`
- Create: `services/control-plane/internal/sandbox/model.go`, `service.go`, `state_machine.go`
- Create: `services/control-plane/internal/lifecycle/service.go`
- Create: `services/control-plane/internal/streaming/hub.go`
- Create: `services/control-plane/internal/storage/object_store.go`
- Create: `services/control-plane/internal/metadata/repository.go`
- Create: `services/control-plane/internal/aws/ecs/provisioner.go`, `aws/s3/store.go`, `aws/dynamodb/repository.go`
- Create: `services/control-plane/internal/runtime/client.go`
- Create: `services/control-plane/internal/auth/service.go`, `middleware.go`
- Create: `services/control-plane/internal/api/server.go`, `errors.go`
- Create: `services/control-plane/tests/fakes.go`, `internal/sandbox/state_machine_test.go`, `internal/sandbox/service_test.go`
- Create: `services/control-plane/README.md`

**Interfaces:**

```go
type SandboxRepository interface {
    Create(ctx context.Context, sandbox Sandbox) error
    Get(ctx context.Context, id string) (Sandbox, error)
    List(ctx context.Context, ownerID string, cursor string, limit int) (Page[Sandbox], error)
    UpdateState(ctx context.Context, id string, expected State, next State, reason string) error
    SaveTaskEndpoint(ctx context.Context, id string, task TaskRef) error
}

type ComputeProvisioner interface {
    Start(ctx context.Context, request ProvisionRequest) (TaskRef, error)
    Stop(ctx context.Context, taskARN string) error
    Describe(ctx context.Context, taskARN string) (TaskStatus, error)
}

type RuntimeClient interface {
    WaitReady(ctx context.Context, endpoint, token string) error
    StartCommand(ctx context.Context, endpoint, token string, request CommandRequest) (CommandResult, error)
    Events(ctx context.Context, endpoint, token, commandID, lastEventID string) (<-chan CommandEvent, error)
    ReadFile(ctx context.Context, endpoint, token, path string) (io.ReadCloser, error)
    WriteFile(ctx context.Context, endpoint, token, path string, body io.Reader) error
    ListFiles(ctx context.Context, endpoint, token, path string) ([]FileEntry, error)
    DeleteFile(ctx context.Context, endpoint, token, path string) error
    ExportWorkspace(ctx context.Context, endpoint, token string) (io.ReadCloser, ArchiveInfo, error)
    RestoreWorkspace(ctx context.Context, endpoint, token string, archive io.Reader) error
}
```

The Go service defines the non-generated domain shapes used by those ports in `services/control-plane/internal/sandbox/model.go`:

```go
type State string
type SandboxID string
type CommandID string
type SnapshotID string

type Page[T any] struct {
    Items      []T
    NextCursor string
}

type Sandbox struct {
    ID              SandboxID
    OwnerID         string
    State           State
    Config          SandboxConfig
    Repository      *RepositoryConfig
    Task            *TaskRef
    CreatedAt       time.Time
    ExpiresAt       time.Time
    LastActivityAt  time.Time
    CurrentCommand  *CommandID
    SnapshotIDs     []SnapshotID
    FailureReason   string
}

type SandboxConfig struct {
    Image                      string
    CPUMillis                  int
    MemoryMiB                  int
    StorageGiB                 int
    MaxLifetime                time.Duration
    DefaultCommandTimeout      time.Duration
    Environment                map[string]string
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
    Command          string
    CWD              string
    Environment      map[string]string
    Timeout          time.Duration
}

type CommandResult struct {
    ID          CommandID
    SandboxID   SandboxID
    Command     string
    ExitCode    *int
    StartedAt   time.Time
    FinishedAt  *time.Time
    TimedOut    bool
    Signal      string
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

type AuthService interface {
    Authenticate(ctx context.Context, apiKey string) (Principal, error)
}

type Principal struct {
    OwnerID string
    KeyID   string
}
```

- Produces `SandboxService.Create`, `Get`, `List`, `Destroy`, `Execute`, `SubscribeEvents`, `ReadFile`, `WriteFile`, `ListFiles`, `DeleteFile`, `CreateSnapshot`, and `Restore` methods.
- Produces typed adapters for ECS, S3, DynamoDB, and the runtime so handlers depend only on interfaces.

- [ ] **Step 1: Write failing state-machine tests.** Assert each allowed transition, reject transitions from `destroyed`, reject commands unless state is `running`, and make repeated destroy idempotent.
- [ ] **Step 2: Implement domain models and transition validation.** Keep state mutation in `state_machine.go`; return `invalid_state_transition` with current and requested states.
- [ ] **Step 3: Write failing service tests with fakes.** Cover create persistence before provisioning, provisioning failure cleanup, runtime readiness failure, command routing, expiration calculation, and missing sandbox behavior.
- [ ] **Step 4: Implement the orchestration service.** Persist `requested`, transition to `provisioning`, call `ComputeProvisioner`, persist the task reference, transition to `starting`, wait for runtime readiness, then transition to `running`.
- [ ] **Step 5: Implement auth and configuration.** Parse required settings, reject production startup without a signing secret and AWS configuration, hash API keys at rest, compare hashes in constant time, and generate a request ID for every request.
- [ ] **Step 6: Implement fake adapters.** Provide deterministic in-memory repository, fake compute provisioner, fake S3 object store, and fake runtime client for all local tests. The fake provisioner must expose task start and stop events to assertions.
- [ ] **Step 7: Run Go tests.** Run `go test ./services/control-plane/...` and `go vet ./services/control-plane/...`; commit as `feat: add control plane domain and service ports`.

### Task 5: Expose the public Go API and SSE event stream

**Files:**

- Modify: `services/control-plane/internal/api/server.go`
- Create: `services/control-plane/internal/api/handlers_sandboxes.go`
- Create: `services/control-plane/internal/api/handlers_commands.go`
- Create: `services/control-plane/internal/api/handlers_files.go`
- Create: `services/control-plane/internal/api/handlers_snapshots.go`
- Create: `services/control-plane/internal/api/sse.go`
- Create: `services/control-plane/internal/api/middleware.go`
- Create: `services/control-plane/internal/api/server_test.go`, `sse_test.go`, `api_contract_test.go`
- Modify: `services/control-plane/cmd/server/main.go`

**Interfaces:**

- Produces the routes in the public API surface with JSON and SSE content types.
- Consumes only `SandboxService`, `AuthService`, `RuntimeClient`, and contract-generated request/response types.

- [ ] **Step 1: Write failing handler tests.** Test authenticated create, list, get, destroy, command submission, file read/write, snapshot creation, restore, request IDs, malformed JSON, and state conflicts.
- [ ] **Step 2: Implement sandbox handlers.** Validate limits before creating a sandbox, enforce the authenticated owner scope, return `202` for asynchronous lifecycle work, and include `Location` when a resource is created.
- [ ] **Step 3: Implement command handlers and SSE.** Return a command ID immediately, stream event IDs in order, flush each event, honor `Last-Event-ID`, send heartbeats every 15 seconds, and close after a terminal event.
- [ ] **Step 4: Implement filesystem handlers.** Stream file downloads, accept bounded request bodies, require the `path` query parameter, normalize paths through the runtime client, and return `204` for successful deletion.
- [ ] **Step 5: Implement snapshot handlers.** Transition the sandbox to `snapshotting`, upload the runtime archive under `sandboxes/{sandboxId}/snapshots/{snapshotId}.tar.zst`, persist metadata only after the object checksum is verified, and restore only into a `running` sandbox with a matching snapshot.
- [ ] **Step 6: Add OpenAPI conformance tests.** Exercise the router against the canonical contract and fail on undocumented status codes or response fields.
- [ ] **Step 7: Boot and verify the service.** Run `go run ./services/control-plane/cmd/server`, then use `curl` to verify `/healthz`, `/readyz`, authenticated sandbox creation through the fake provider, an SSE stream, and graceful shutdown. Commit as `feat: expose sandbox api and live events`.

### Task 6: Separate lifecycle and snapshot worker boundaries

**Files:**

- Create: `services/lifecycle-manager/go.mod`, `cmd/worker/main.go`, `internal/expiration/worker.go`, `internal/reconcile/reconciler.go`, `README.md`
- Create: `services/snapshot-service/go.mod`, `cmd/worker/main.go`, `internal/service/service.go`, `internal/archive/validator.go`, `README.md`
- Create: `services/lifecycle-manager/internal/worker/worker_test.go`, `services/snapshot-service/internal/service/service_test.go`
- Modify: `go.work`, `Makefile`

**Interfaces:**

```go
type LifecycleWorker interface {
    ExpireDueSandboxes(ctx context.Context, now time.Time, batchSize int) (int, error)
    ReconcileTasks(ctx context.Context, now time.Time, batchSize int) (int, error)
}

type SnapshotService interface {
    Create(ctx context.Context, sandboxID string) (SnapshotMetadata, error)
    Restore(ctx context.Context, sandboxID, snapshotID string) error
    DeleteExpired(ctx context.Context, now time.Time, batchSize int) (int, error)
}
```

- [ ] **Step 1: Write worker tests.** Verify an expired sandbox is stopped once, an orphaned task is stopped, a stale `starting` sandbox becomes `failed` after its deadline, and a second worker run does not duplicate side effects.
- [ ] **Step 2: Implement lifecycle polling.** Query due sandboxes from the repository, call the control-plane service for legal transitions, and use bounded batches with structured metrics.
- [ ] **Step 3: Write snapshot service tests.** Verify archive checksum mismatch does not create available metadata, restore rejects an unknown snapshot, and expired metadata produces a delete request for the correct S3 key.
- [ ] **Step 4: Implement the snapshot boundary.** Keep S3 archive naming, checksum validation, metadata transitions, and retention policy in this service; call it from the control plane through an interface.
- [ ] **Step 5: Boot worker binaries.** Add a process that runs once with `--once` for CI and a bounded polling loop for AWS deployment. Run `go test ./services/lifecycle-manager/... ./services/snapshot-service/...`; commit as `feat: add lifecycle and snapshot workers`.

### Task 7: Build the sandbox images and local container workflow

**Files:**

- Create: `images/sandbox-base/Dockerfile`, `images/sandbox-base/entrypoint.sh`, `images/sandbox-base/README.md`
- Create: `images/sandbox-dev/Dockerfile`, `images/sandbox-dev/README.md`
- Create: `infrastructure/docker/compose.yaml`, `infrastructure/docker/README.md`
- Create: `scripts/build-images.sh`, `scripts/smoke-runtime.sh`

**Interfaces:**

- Produces an ECR-ready image tagged with an immutable Git SHA and a runtime container listening on port `8080`.
- Produces a local compose profile for control plane, sandbox runtime, and test dependencies without changing the public product abstraction.

- [ ] **Step 1: Write the image smoke test.** Build the image and assert `git`, `curl`, `wget`, `bash`, `python3`, `pip3`, `node`, `npm`, `go`, `rustc`, `cargo`, `gcc`, `g++`, and `make` return versions.
- [ ] **Step 2: Implement the base image.** Use a pinned Debian/Ubuntu base digest, install the listed development tools, create a non-root `sandbox` user, create `/workspace`, copy the runtime binary, and set the runtime as the entrypoint.
- [ ] **Step 3: Add runtime hardening defaults.** Do not add privileged mode, host mounts, Docker socket access, or cloud credentials. Set a writable workspace and temporary directory, and make the container fail closed when the runtime token is absent.
- [ ] **Step 4: Build the dev image and compose file.** Use the same runtime contract, expose the runtime only to the compose network, and configure fake adapters for control-plane integration tests.
- [ ] **Step 5: Run the image checks.** Run `docker build`, `scripts/smoke-runtime.sh`, and `docker compose -f infrastructure/docker/compose.yaml config`; commit as `build: add sandbox images and local containers`.

### Task 8: Provision the AWS development environment with Terraform

**Files:**

- Create: `infrastructure/terraform/versions.tf`, `providers.tf`, `variables.tf`, `outputs.tf`
- Create: `infrastructure/terraform/modules/networking/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/ecs/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/ecr/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/s3/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/dynamodb/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/cloudwatch/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/iam/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/control-plane/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/modules/sandbox/{main.tf,variables.tf,outputs.tf}`
- Create: `infrastructure/terraform/environments/dev/{main.tf,variables.tf,terraform.tfvars.example}`
- Create: `infrastructure/terraform/environments/prod/{main.tf,variables.tf,terraform.tfvars.example}`
- Create: `infrastructure/aws/console-verification.md`, `infrastructure/aws/deployment-runbook.md`
- Create: `infrastructure/environments/dev.env.example`, `infrastructure/environments/prod.env.example`

**Interfaces:**

- Produces Terraform outputs for VPC ID, private subnet IDs, ECS cluster ARN, control-plane service name, sandbox task definition ARN, ECR repositories, snapshot bucket, DynamoDB table, log groups, and security-group IDs.
- Produces an ECS task launch input with image URI, CPU, memory, environment, runtime token, sandbox ID, and expiration metadata.

- [ ] **Step 1: Write Terraform validation checks.** Validate required tags, one AWS region, non-empty ECR image URIs, bounded CPU/memory combinations, S3 public access block, DynamoDB point-in-time recovery, and log retention.
- [ ] **Step 2: Implement networking.** Create a VPC across two availability zones, public subnets for the ALB, private subnets for control-plane and sandbox tasks, NAT for outbound package/Git traffic, route tables, and security groups with least-privilege ingress.
- [ ] **Step 3: Implement storage and observability.** Create an encrypted S3 bucket with versioning and lifecycle expiration, a DynamoDB table keyed by `sandboxId` with a GSI for owner plus creation time, CloudWatch log groups, and metric alarms for task launch failures and control-plane errors.
- [ ] **Step 4: Implement ECR and IAM.** Create separate control-plane and sandbox repositories, enable image scanning on push, give the control-plane task role only ECS, S3, DynamoDB, and CloudWatch permissions it needs, and give the sandbox task execution role only image pull and log delivery permissions.
- [ ] **Step 5: Implement ECS services.** Create the public ALB and control-plane service. Create a sandbox task definition with `awsvpc`, no public IP, non-root runtime, CPU/memory variables, CloudWatch logging, and a security group allowing runtime port `8080` only from the control-plane group.
- [ ] **Step 6: Add dev and prod compositions.** Make dev use smaller bounded defaults and prod require explicit image digests and deletion protection settings. Keep prod configuration syntactically valid without applying it in the hackathon.
- [ ] **Step 7: Run infrastructure checks.** Run `terraform fmt -check -recursive`, `terraform init -backend=false`, `terraform validate` in both environments, and `terraform plan` for dev with example variables. Commit as `infra: define aws sandbox platform`.

### Task 9: Implement real AWS adapters and task reconciliation

**Files:**

- Modify: `services/control-plane/internal/aws/ecs/provisioner.go`
- Modify: `services/control-plane/internal/aws/s3/store.go`
- Modify: `services/control-plane/internal/aws/dynamodb/repository.go`
- Create: `services/control-plane/internal/aws/ecs/task_definition.go`
- Create: `services/control-plane/internal/aws/dynamodb/codec.go`
- Create: `services/control-plane/internal/aws/retry.go`
- Create: `services/control-plane/internal/aws/aws_integration_test.go`
- Modify: `services/control-plane/internal/config/config.go`
- Modify: `services/control-plane/cmd/server/main.go`

**Interfaces:**

- `ComputeProvisioner.Start` calls `ecs.RunTask`, waits for the task network interface, and returns task ARN, private IP, endpoint, and runtime token reference.
- `ComputeProvisioner.Stop` calls `ecs.StopTask` and treats an already-stopped task as success.
- `ObjectStore.Put` accepts a stream, content length, checksum, and metadata; `ObjectStore.Get` returns a stream; `ObjectStore.Delete` is idempotent.
- `SandboxRepository` maps DynamoDB conditional writes to typed conflict errors.

- [ ] **Step 1: Write adapter tests around AWS SDK fakes.** Cover ECS launch failure, missing network attachment, stop of a missing task, S3 checksum mismatch, DynamoDB conditional update conflict, and retryable throttling.
- [ ] **Step 2: Implement ECS provisioning.** Pass sandbox tags on every task, use a dedicated task definition, wait with a deadline, inspect `awsvpc` attachments for the private IP, and persist no credentials in task environment variables.
- [ ] **Step 3: Implement DynamoDB persistence.** Use conditional expressions for expected-state updates, store timestamps as RFC 3339 strings, keep owner and expiration attributes queryable, and map absent items to `sandbox_not_found`.
- [ ] **Step 4: Implement S3 streaming.** Use multipart upload for archives, set server-side encryption, attach sandbox and snapshot metadata, verify the returned checksum, and never make the bucket public.
- [ ] **Step 5: Add bounded retries.** Retry only throttling and transient network errors with exponential backoff and jitter; do not retry validation, authorization, state conflicts, or terminal ECS failures.
- [ ] **Step 6: Run adapter tests and a dev smoke test.** Run unit tests with fake clients, then deploy to the Terraform dev environment and verify one task appears in ECS, one log stream appears in CloudWatch, and one metadata item appears in DynamoDB. Commit as `feat: connect control plane to aws adapters`.

### Task 10: Add the TypeScript developer SDK

**Files:**

- Create: `packages/sdk-typescript/package.json`, `src/index.ts`, `src/config.ts`, `src/client.ts`
- Create: `packages/sdk-typescript/src/sandbox.ts`, `src/commands.ts`, `src/filesystem.ts`, `src/snapshots.ts`, `src/events.ts`, `src/errors.ts`, `src/types.ts`
- Create: `packages/sdk-typescript/test/client.test.ts`, `test/sandbox.test.ts`, `test/events.test.ts`, `test/errors.test.ts`
- Create: `packages/sdk-typescript/README.md`, `packages/sdk-typescript/tsconfig.json`

**Interfaces:**

```typescript
export class SandboxClient {
  constructor(config: SandboxClientConfig);
  sandboxes: SandboxCollection;
}

export interface SandboxCollection {
  create(config?: CreateSandboxInput): Promise<SandboxHandle>;
  get(id: string): Promise<SandboxHandle>;
  list(options?: ListSandboxesOptions): Promise<Page<SandboxSummary>>;
}

export interface SandboxHandle {
  readonly id: string;
  get(): Promise<SandboxSummary>;
  exec(command: string, options?: ExecOptions): Promise<CommandResult>;
  execStream(command: string, options?: ExecOptions): AsyncIterable<CommandEvent>;
  readFile(path: string): Promise<string>;
  writeFile(path: string, content: string | Uint8Array): Promise<void>;
  listFiles(path?: string): Promise<FileEntry[]>;
  deleteFile(path: string): Promise<void>;
  snapshot(options?: SnapshotOptions): Promise<SnapshotMetadata>;
  restore(snapshotId: string): Promise<void>;
  destroy(): Promise<void>;
}

export interface SandboxClientConfig {
  baseUrl: string;
  apiKey: string;
  fetch?: typeof fetch;
  requestTimeoutMs?: number;
}

export interface CreateSandboxInput {
  image?: string;
  cpuMillis?: number;
  memoryMiB?: number;
  storageGiB?: number;
  maxLifetimeSeconds?: number;
  defaultCommandTimeoutSeconds?: number;
  environment?: Record<string, string>;
  repository?: RepositoryConfig;
  snapshotId?: string;
}

export interface ExecOptions {
  cwd?: string;
  environment?: Record<string, string>;
  timeoutSeconds?: number;
}

export interface SnapshotOptions {
  expiresInSeconds?: number;
}

export interface ListSandboxesOptions {
  cursor?: string;
  limit?: number;
  state?: SandboxState;
}

export interface Page<T> {
  items: T[];
  nextCursor?: string;
}

type SandboxSummary = Sandbox;
```

- [ ] **Step 1: Write failing SDK tests with a fake fetch.** Assert request URLs, headers, JSON bodies, typed error parsing, `202` command handling, SSE event ordering, and `Last-Event-ID` reconnect behavior.
- [ ] **Step 2: Implement configuration and HTTP client.** Require `baseUrl`, accept an injected `fetch`, send `Authorization: Bearer`, add an idempotency key for create and snapshot, and expose request IDs on `SandboxError`.
- [ ] **Step 3: Implement the sandbox handle.** Keep methods product-focused and route every operation through the generated API types; do not expose ECS, Fargate, S3, DynamoDB, or IAM names. Make the resulting handle usable by the dashboard and the thin MCP adapter.
- [ ] **Step 4: Implement streaming.** Parse SSE frames with multiline data support, emit typed events, reconnect once on a dropped connection using the last event ID, and stop after `completed` or `failed`.
- [ ] **Step 5: Add package quality gates.** Configure declaration output, ESM plus CJS-compatible publishing if required by the package toolchain, source maps, API surface tests, and a README example matching the PRD flow.
- [ ] **Step 6: Run SDK tests.** Run `pnpm --filter @haedes/sdk test`, `pnpm --filter @haedes/sdk typecheck`, and `pnpm --filter @haedes/sdk build`; commit as `feat: add typescript sandbox sdk`.

### Task 11: Create the public web and dashboard applications

The dashboard is the human observability and control surface. It must support the agent-led demo and make lifecycle, commands, snapshots, AWS execution status, errors, and request IDs legible, but the primary user flow begins in the coding agent rather than in the dashboard.

**Files:**

- Create: `apps/web/package.json`, `app/layout.tsx`, `app/page.tsx`, `app/globals.css`, `README.md`
- Create: `apps/dashboard/package.json`, `app/layout.tsx`, `app/page.tsx`, `app/sandboxes/page.tsx`, `app/sandboxes/[id]/page.tsx`, `app/globals.css`
- Create: `apps/dashboard/src/lib/api-client.ts`, `src/lib/query-keys.ts`, `src/components/sandbox-list.tsx`, `src/components/create-sandbox-form.tsx`, `src/components/sandbox-header.tsx`, `src/components/terminal.tsx`, `src/components/execution-timeline.tsx`, `src/components/snapshot-panel.tsx`, `src/components/aws-status.tsx`
- Create: `apps/dashboard/test/sandbox-list.test.tsx`, `test/terminal.test.tsx`, `playwright/sandbox-flow.spec.ts`
- Create: `docs/api/dashboard-data-flow.md`

**Interfaces:**

- Produces a dashboard API client that wraps `SandboxClient` or the generated API client and exposes `listSandboxes`, `createSandbox`, `getSandbox`, `executeCommand`, `subscribeToCommand`, `createSnapshot`, `restoreSnapshot`, and `destroySandbox`.
- Produces UI states for loading, empty, provisioning, running, snapshotting, stopping, failed, and destroyed sandboxes. No screen reads mock data directly.

- [ ] **Step 1: Write failing component tests.** Cover empty sandbox list, a provisioning card, command output appended in order, failed command styling, snapshot availability, restore action, and destroy confirmation.
- [ ] **Step 2: Build the public landing page.** Explain the brain/computer distinction, show the AWS lifecycle, link to the dashboard, and include a short code sample that uses the SDK abstraction.
- [ ] **Step 3: Build the sandbox list.** Render ID, state, repository, creation time, expiry, resource configuration, and a link to details. Poll only while a sandbox is non-terminal.
- [ ] **Step 4: Build the detail view.** Add a create/execute workflow, live terminal, command history, lifecycle timeline, snapshot history, AWS status, and destroy action. Show errors with request IDs.
- [ ] **Step 5: Wire SSE into the terminal.** Subscribe when a command starts, append stdout/stderr without losing line breaks, reconnect from the last event ID, and close on completion.
- [ ] **Step 6: Add real API configuration.** Read the control-plane URL from a public environment variable, use an explicit development API key, and show a configuration error instead of silently switching to mock data.
- [ ] **Step 7: Run UI tests.** Run `pnpm --filter web build`, `pnpm --filter dashboard test`, and `pnpm --filter dashboard exec playwright test`; commit as `feat: add web and sandbox dashboard`.

### Task 12: Add GitHub and MCP coding-agent integration

**Files:**

- Create: `integrations/github/src/repository-url.ts`, `src/credentials.ts`, `src/clone-request.ts`, `test/repository-url.test.ts`, `README.md`
- Create: `integrations/mcp/package.json`, `src/server.ts`, `src/tools.ts`, `src/validation.ts`, `src/platform-client.ts`, `src/errors.ts`, `test/tools.test.ts`, `test/validation.test.ts`, `test/errors.test.ts`, `test/server.test.ts`, `README.md`
- Modify: `integrations/mcp/src/tool-contract.schema.json` with the concrete tool definitions and result examples
- Modify: `integrations/agents/src/agent-adapter.ts`, `src/workspace-session.ts`, `test/agent-adapter.test.ts`, `README.md` to document only provider-neutral host contracts; do not build a custom agent loop
- Create: `integrations/examples/basic-agent/package.json`, `src/main.ts`, `README.md`
- Create: `integrations/examples/coding-agent/package.json`, `src/main.ts`, `README.md`
- Create: `examples/basic-agent/README.md`, `examples/coding-agent/README.md`, `examples/github-fixer/README.md`
- Create: `internal/fixtures/auth-bug-repository/README.md`, `package.json`, `src/auth.ts`, `test/auth.test.ts`

**Interfaces:**

```typescript
export interface AgentSandboxSession {
  create(input: CreateSandboxInput): Promise<SandboxHandle>;
  run(command: string, options?: ExecOptions): AsyncIterable<CommandEvent>;
  read(path: string): Promise<string>;
  write(path: string, content: string): Promise<void>;
  save(): Promise<SnapshotMetadata>;
  close(): Promise<void>;
}
```

`AgentSandboxSession` is a thin host-facing session facade for tests and examples; it is not an agent planner, model wrapper, or replacement for the real coding-agent client.

The MCP server is the implementation target for the hackathon’s agent-native layer. It uses the TypeScript SDK, which in turn uses the canonical HTTP API, unless a shared direct HTTP client is demonstrably simpler. It must not call AWS, ECS, Fargate, S3, DynamoDB, CloudWatch, or the Rust runtime directly. It must not persist sandbox lifecycle state or define a second state machine.

Required MCP tools and responsibilities:

```text
sandbox_create          -> create a computer and return sandbox identity/state
sandbox_get/list        -> inspect one or many computers
sandbox_exec            -> submit a command and return command identity/result
sandbox_exec_stream     -> expose ordered command events
sandbox_read_file       -> read bounded file content
sandbox_write_file      -> write bounded file content
sandbox_list_files      -> list workspace entries
sandbox_delete_file     -> delete a workspace entry
sandbox_snapshot        -> save workspace and return snapshot identity
sandbox_restore         -> restore a snapshot into a fresh/running sandbox
sandbox_destroy         -> request idempotent computer destruction
```

- [ ] **Step 1: Write repository URL tests.** Accept HTTPS GitHub URLs and `owner/repository` shorthand, normalize `.git`, reject non-GitHub hosts for this integration, and never log embedded credentials.
- [ ] **Step 2: Implement GitHub request construction.** Keep tokens in an injected credential provider, pass clone credentials through a runtime-safe mechanism, redact token-shaped values from command and event logs, and return a typed `github_repository_invalid` error.
- [ ] **Step 3: Run the real-agent compatibility spike.** Select one existing coding-agent client, preferred examples being Claude Code or Codex if the current client can consume the MCP integration cleanly. Verify the exact MCP configuration, tool discovery, streaming behavior, authentication boundary, and non-interactive demo invocation in the actual environment. Do not document vendor support until this passes; keep the adapter generic if the preferred client is unavailable.
- [ ] **Step 4: Write the MCP contract and failing adapter tests.** Validate tool names and schemas, required and bounded inputs, structured results, sandbox and command IDs, typed platform errors, authentication failures, nonexistent sandboxes, and rejection of AWS credentials or lifecycle-only fields.
- [ ] **Step 5: Implement the MCP server.** Translate tool calls into SDK/platform operations, preserve idempotency and HTTP error semantics, return agent-friendly summaries, stream or expose command execution appropriately, and keep the server-side API key outside model-visible inputs and outputs.
- [ ] **Step 6: Test every platform operation through MCP.** Cover create, get, list, command submit, command stream, read, write, list files, delete file, snapshot, restore, and destroy with a fake control plane. Assert that MCP has no direct AWS dependency and no independent persistence.
- [ ] **Step 7: Write and run the real coding-agent journey.** From the chosen coding agent, ask it to fix `internal/fixtures/auth-bug-repository`. Verify the agent creates a sandbox, clones the repository, installs dependencies, runs the failing test, reads and edits files, reruns to pass, snapshots, destroys the original task, restores into a new task, reruns to pass, and destroys the new task. Commit as `feat: add mcp coding agent integration`.

### Task 13: Add integration, end-to-end, security, and load tests

**Files:**

- Create: `tests/integration/control-plane-runtime.test.ts`
- Create: `tests/integration/snapshot-roundtrip.test.ts`
- Create: `tests/integration/mcp-platform.test.ts`
- Create: `tests/e2e/sdk-coding-agent.spec.ts`, `tests/e2e/mcp-coding-agent.spec.ts`, `tests/e2e/dashboard-flow.spec.ts`
- Create: `tests/security/path-traversal.test.ts`, `tests/security/auth-boundary.test.ts`, `tests/security/resource-limits.test.ts`, `tests/security/archive-extraction.test.ts`
- Create: `tests/load/command-stream.js`, `tests/load/sandbox-lifecycle.js`
- Create: `tests/README.md`, `tests/fixtures/large-file.bin` generated by a script rather than committed as opaque data

**Interfaces:**

- Produces a repeatable local test command using fake AWS adapters and a real Rust runtime container.
- Produces a gated AWS smoke test that is opt-in through `AWS_INTEGRATION_TESTS=true` and never runs against production by default.

- [ ] **Step 1: Write the control-plane/runtime integration test.** Create a fake sandbox, wait for running, execute `printf`, read and write a file, list the workspace, delete the file, and assert the event stream and command result agree.
- [ ] **Step 2: Write the snapshot round-trip test.** Create a file and fixture repository, export it, verify checksum and metadata, destroy the original task, restore into a fresh fake task, and assert file contents plus test output survive.
- [ ] **Step 3: Write security tests.** Verify unauthenticated public API calls fail, runtime tokens cannot access another sandbox, path traversal and symlink escapes fail, oversized bodies fail, command timeouts kill descendants, and unsafe archive entries are rejected.
- [ ] **Step 4: Test the MCP boundary.** Assert tool schema validity, create/exec/read/write/snapshot/destroy success through a fake control plane, typed error propagation, authentication failures, and nonexistent sandbox references. Assert no tool can supply AWS credentials or bypass the public API.
- [ ] **Step 5: Write browser and agent journeys.** Assert the dashboard creates a sandbox, shows lifecycle transitions, streams output, displays the execution timeline, creates a snapshot, destroys the old sandbox, restores into a new sandbox, and shows the passing test. Run the chosen real coding agent against the same fixture as an opt-in end-to-end test, recording the verified client/version and MCP configuration.
- [ ] **Step 6: Add load scenarios.** Exercise concurrent command streams and repeated create/destroy requests with bounded virtual users. Record command latency, event delivery delay, task launch latency, and error rate; do not make a production performance claim from local numbers.
- [ ] **Step 7: Run all tests.** Run `make test`, `pnpm exec playwright test`, the MCP integration suite, and `k6 run tests/load/command-stream.js` against local services; commit as `test: cover sandbox journeys and boundaries`.

### Task 14: Add CI, release checks, and operational documentation

**Files:**

- Create: `.github/workflows/ci.yml`, `.github/workflows/contracts.yml`, `.github/workflows/containers.yml`, `.github/workflows/terraform.yml`, `.github/workflows/security.yml`
- Create: `.github/dependabot.yml`, `.github/pull_request_template.md`
- Modify: `README.md`, `CONTRIBUTING.md`, `SECURITY.md`
- Create: `docs/architecture/system.md`, `docs/architecture/data-flow.md`, `docs/sandbox/runtime.md`, `docs/sandbox/limits.md`, `docs/aws/dev-deployment.md`, `docs/aws/demo-runbook.md`, `docs/security/threat-model.md`, `docs/decisions/0002-fargate-task-per-sandbox.md`, `docs/decisions/0003-s3-workspace-archives.md`
- Create: `scripts/doctor.sh`, `scripts/demo.sh`, `scripts/verify-aws-resources.sh`

**Interfaces:**

- Produces one pull-request gate that runs formatting, type checking, contract generation, Go tests/vet, Rust fmt/clippy/test, image smoke tests, Terraform validation, and security tests.
- Produces an operator runbook that can answer: how the chosen coding agent connects through MCP, what was created in AWS, how to find a sandbox task, how to inspect CloudWatch logs, how to locate an S3 snapshot, how to stop an orphan, and how to clean the dev environment.

- [ ] **Step 1: Write CI scripts locally first.** Run every CI command through `make check` or a named script so a developer can reproduce failures without reading workflow YAML.
- [ ] **Step 2: Add the CI workflow.** Use least-privilege GitHub permissions, cache pnpm/Go/Cargo dependencies, run jobs in parallel, upload test reports and screenshots, and make contract drift fail the build.
- [ ] **Step 3: Add container and Terraform workflows.** Build images on pull requests without pushing, scan images, run Terraform fmt/validate/plan with no cloud mutation, and require an environment-scoped credential for any apply job.
- [ ] **Step 4: Add security automation.** Run dependency audits, secret scanning, IaC scanning, and the repository’s path/auth/archive tests. Fail on committed secrets or publicly readable S3 resources.
- [ ] **Step 5: Write the demo runbook.** Make the primary script start in the verified coding-agent client, configure the MCP server, ask it to fix the fixture repository, and use the dashboard only as the human observability surface. Include exact commands to deploy the image, apply dev Terraform, and verify ECS, CloudWatch, S3, DynamoDB, and task destruction in the AWS console. Keep curl as a debugging and infrastructure-verification path.
- [ ] **Step 6: Run a clean checkout rehearsal.** Clone the repository into a temporary directory, follow `CONTRIBUTING.md`, run `scripts/doctor.sh`, run `make check`, and execute the demo script with a fake provider. Commit as `ci: add repository quality and demo workflows`.

### Task 15: Perform the release rehearsal and scope audit

**Files:**

- Modify: `README.md`, `docs/aws/demo-runbook.md`, `docs/api/public-api.md`, and any source file found by the checks below
- Create: `docs/decisions/0004-hackathon-scope.md`

**Interfaces:**

- Produces a single verified command path and a written list of what is live, what is intentionally deferred, and what a judge can see in the three-minute demo.

- [ ] **Step 1: Run the vertical slice from a clean environment.** Start in the verified coding-agent client, invoke the MCP tools to create a sandbox, clone the fixture repository, run the failing test, make the fix, rerun the test, create an S3 snapshot, destroy the task, restore into a new task, rerun the test, and destroy the new task. Repeat the same operation through the SDK as a client-level check.
- [ ] **Step 2: Verify AWS evidence.** Capture the ECS task lifecycle, CloudWatch command logs, S3 snapshot object, DynamoDB state record, and stopped ECS task. Confirm no public S3 object and no sandbox task credentials.
- [ ] **Step 3: Exercise failure paths.** Force a command timeout, runtime health failure, S3 checksum failure, duplicate create request, duplicate destroy request, and restore into a non-running sandbox. Confirm typed errors and cleanup.
- [ ] **Step 4: Audit the plan against the PRD.** Confirm every must-build item maps to a task and every do-not-prioritize item remains out of the implementation. Record gaps with an owner and a concrete next milestone rather than expanding the hackathon slice.
- [ ] **Step 5: Produce the submission-ready demo.** Keep the video centered on one coding-agent bug-fix story: agent decision, MCP tool call, requested/provisioning/starting/running lifecycle, real command stream, failed test, file edit, passing retest, snapshot, task destruction, new task, restore, passing retest, and final destruction. Briefly show ECS, CloudWatch, S3, and task shutdown, then return to the agent/product before the three-minute limit. Commit any final documentation corrections as `docs: finalize hackathon release plan`.

## Hackathon critical path

The implementation order is intentionally narrow and infrastructure-first:

```text
Rust runtime
        |
        v
Go control plane
        |
        v
real Fargate sandbox
        |
        v
command execution and streaming
        |
        v
filesystem
        |
        v
snapshot and restore
        |
        v
MCP adapter
        |
        v
one verified real coding agent
        |
        v
dashboard
        |
        v
three-minute demo
```

The SDK and HTTP contract are built early because MCP, the dashboard, and customer-facing examples depend on them, but the visible demo starts in the coding agent. Infrastructure and product depth take priority over broad agent-framework coverage.

## Data flow

```text
Claude Code / Codex / custom coding agent
        |
        v
MCP adapter
        |
        +--------------------+
        |                    |
        v                    v
TypeScript SDK        HTTP API /v1
        |                    |
        +----------+---------+
                   v
           Public ALB -> Go control plane
                |
                +-> DynamoDB: sandbox state, commands, snapshots
                +-> ECS: start/stop/describe one Fargate task
                +-> Runtime HTTP: exec, files, snapshot export/restore
                +-> S3: compressed workspace archive
                +-> CloudWatch: control-plane and runtime logs
                                      |
                                      v
                              Rust runtime in task
                                |          |
                                v          v
                           /workspace  child process group
```

Command flow:

1. The coding agent invokes an MCP tool. MCP validates the input and delegates through the SDK or shared HTTP client; the resulting client sends `POST /v1/sandboxes/{id}/commands` with a bounded command, working directory, environment, and timeout.
2. The control plane verifies ownership and `running` state, records the command, and asks the runtime to start it.
3. The runtime creates a process group, publishes stdout/stderr events, kills the group when the deadline expires, and emits a terminal result.
4. The control plane persists the result and broadcasts events to the SSE subscriber.
5. The SDK and dashboard can reconnect using the last event ID while the command record is retained.

Snapshot flow:

1. The caller asks for a snapshot with an idempotency key.
2. The control plane transitions `running -> snapshotting` and asks the runtime to export `/workspace`.
3. The snapshot service streams the archive to encrypted S3, calculates/verifies SHA-256, and persists available metadata.
4. The control plane transitions `snapshotting -> running` only after metadata is available.
5. Restore starts or targets a fresh running task, validates the archive, extracts only safe workspace entries, and records the snapshot ID on the new sandbox.

## Security model

- Public API authentication uses hashed API keys with owner scoping. Production requires authentication; local fake-provider tests use a clearly named test key.
- Every request receives a request ID. Logs include request ID, sandbox ID, command ID, and snapshot ID, but not API keys, GitHub tokens, environment values, or file contents.
- Runtime tokens are random, sandbox-scoped, short-lived, stored outside public responses, and accepted only on the matching task endpoint.
- ECS tasks run as non-root, with no privileged flag, no host filesystem, no Docker socket, no AWS credentials, and a task security group that accepts runtime traffic only from the control plane.
- Command execution is intentionally powerful inside the sandbox. Safety comes from task isolation, no credentials, bounded CPU/memory/storage/lifetime, command timeout, output cap, and destroy/reconcile cleanup.
- File paths are resolved under `/workspace`; all archive extraction is validated against traversal, absolute paths, symlink escapes, and duplicate conflicting entries.
- S3 blocks public access, uses encryption and lifecycle rules, and does not return raw object credentials to SDK users.
- GitHub integration accepts only approved URL forms in the first slice and keeps credentials in an injected secret boundary.
- Rate limits apply per API key to sandbox creation, command submission, file upload, snapshot creation, and SSE connections. The initial implementation may use in-process limits with a documented single-instance constraint; DynamoDB-backed limits are a later scale step.

## Observability

Emit structured JSON logs and metrics with these fields:

```text
request_id, owner_id, sandbox_id, command_id, snapshot_id,
state, duration_ms, outcome, aws_task_arn
```

Metrics:

- `sandbox_create_total`, `sandbox_create_failure_total`
- `sandbox_ready_seconds`
- `sandbox_destroy_total`, `sandbox_orphan_total`
- `command_total`, `command_duration_seconds`, `command_timeout_total`
- `command_event_delivery_delay_seconds`
- `snapshot_total`, `snapshot_bytes`, `snapshot_failure_total`, `snapshot_restore_total`
- `runtime_health_failure_total`

Health endpoints:

- `/healthz` proves the process is alive and does not call AWS.
- `/readyz` proves configuration is loaded and required dependencies can be reached within a deadline.
- Runtime `/healthz` proves the process can access `/workspace` and report its version.

## Definition of done

- A clean checkout can run `make check` without manual source edits.
- Contract generation is reproducible and CI rejects drift.
- The control plane boots and serves health endpoints.
- The Rust runtime boots and passes filesystem, execution, timeout, streaming, path, and archive tests.
- The dashboard uses the real API client and displays live command output and lifecycle history.
- The SDK supports create, execute, stream, read, write, list, delete, snapshot, restore, and destroy.
- The MCP adapter passes tool schema and boundary tests, delegates every tool to the canonical platform contract, and exposes structured sandbox/command/snapshot identities and typed errors.
- One verified real coding agent can use MCP to complete the fixture bug-fix journey end to end; the exact client/version and setup are recorded without making unsupported vendor claims.
- A real AWS dev deployment creates a Fargate task per sandbox, writes metadata to DynamoDB, logs to CloudWatch, and stores snapshots in encrypted S3.
- Destroying a sandbox stops the ECS task and marks the sandbox terminal only after reconciliation.
- Restoring a snapshot into a fresh sandbox recreates the workspace and preserves the fixture repository’s passing test.
- Security tests cover authentication, task-token isolation, path traversal, symlink escapes, archive traversal, output/body limits, and descendant process termination.
- The demo runbook proves AWS visibility in the video and the README explains why AWS is part of the product rather than only hosting.

## Self-review against the source documents

- Product summary and vision: Tasks 4, 5, 10, and 12 define the computer abstraction, canonical HTTP contract, MCP adapter, and public SDK.
- Temporary sandboxes, command execution, live output, filesystem, Git, dependencies, cleanup, resource controls: Tasks 3, 4, 5, 7, and 9.
- Snapshots, restoration, and agent handoff: Task 6, Task 5 snapshot routes, and Task 12 coding-agent example.
- Dashboard, terminal, execution timeline, AWS status: Task 11.
- AWS compute, storage, metadata, logs, images, networking: Tasks 7 through 9.
- Hackathon must-build list and visible AWS demo: Tasks 12, 14, and 15, with the primary flow beginning in the verified coding agent through MCP.
- Hackathon do-not-prioritize list: Global Constraints and Task 15 scope audit.
- Repository modularity: repository shape plus Tasks 1 through 14, with no generic backend catch-all.
- Security and failure questions: Global Constraints, Task 3, Task 4, Task 6, Task 9, Task 13, and the Security Model.

This plan deliberately leaves no task dependent on an unnamed type, an unowned directory, or an unspecified test command. Any feature not listed in the must-build scope needs a new decision record before implementation.
