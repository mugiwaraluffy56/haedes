# haedes Complete Implementation Tickets

This board is the ticket-level execution breakdown of every task in `PLAN.md`. It is intentionally more granular than the 15 plan tasks: the plan remains the architecture and technical detail, while these tickets are owner-sized units of work.

Each ticket has one owner, explicit blocking edges, and acceptance criteria. To claim a ticket, replace `unassigned` with a teammate’s name and change `blocked` or `ready-for-agent` to `in-progress` when its blockers are complete. Do not treat this board as complete until the traceability table at the end is complete.

## GitHub issue links

Each ticket is tracked by one GitHub issue. Issue titles use the `feat:` convention without ticket-number prefixes; the ticket ID remains the stable planning identifier. Issues were published in dependency order, so a ticket ID and GitHub issue number are not identical for every ticket.

| Ticket | GitHub issue |
| --- | --- |
| 01 | [#1](https://github.com/mugiwaraluffy56/haedes/issues/1) |
| 02 | [#2](https://github.com/mugiwaraluffy56/haedes/issues/2) |
| 03 | [#3](https://github.com/mugiwaraluffy56/haedes/issues/3) |
| 04 | [#4](https://github.com/mugiwaraluffy56/haedes/issues/4) |
| 05 | [#5](https://github.com/mugiwaraluffy56/haedes/issues/5) |
| 06 | [#6](https://github.com/mugiwaraluffy56/haedes/issues/6) |
| 07 | [#7](https://github.com/mugiwaraluffy56/haedes/issues/7) |
| 08 | [#8](https://github.com/mugiwaraluffy56/haedes/issues/8) |
| 09 | [#9](https://github.com/mugiwaraluffy56/haedes/issues/9) |
| 10 | [#10](https://github.com/mugiwaraluffy56/haedes/issues/10) |
| 11 | [#11](https://github.com/mugiwaraluffy56/haedes/issues/11) |
| 12 | [#12](https://github.com/mugiwaraluffy56/haedes/issues/12) |
| 13 | [#13](https://github.com/mugiwaraluffy56/haedes/issues/13) |
| 14 | [#14](https://github.com/mugiwaraluffy56/haedes/issues/14) |
| 15 | [#15](https://github.com/mugiwaraluffy56/haedes/issues/15) |
| 16 | [#16](https://github.com/mugiwaraluffy56/haedes/issues/16) |
| 17 | [#17](https://github.com/mugiwaraluffy56/haedes/issues/17) |
| 18 | [#18](https://github.com/mugiwaraluffy56/haedes/issues/18) |
| 19 | [#24](https://github.com/mugiwaraluffy56/haedes/issues/24) |
| 20 | [#19](https://github.com/mugiwaraluffy56/haedes/issues/19) |
| 21 | [#23](https://github.com/mugiwaraluffy56/haedes/issues/23) |
| 22 | [#20](https://github.com/mugiwaraluffy56/haedes/issues/20) |
| 23 | [#21](https://github.com/mugiwaraluffy56/haedes/issues/21) |
| 24 | [#22](https://github.com/mugiwaraluffy56/haedes/issues/22) |
| 25 | [#25](https://github.com/mugiwaraluffy56/haedes/issues/25) |
| 26 | [#26](https://github.com/mugiwaraluffy56/haedes/issues/26) |
| 27 | [#27](https://github.com/mugiwaraluffy56/haedes/issues/27) |
| 28 | [#28](https://github.com/mugiwaraluffy56/haedes/issues/28) |
| 29 | [#29](https://github.com/mugiwaraluffy56/haedes/issues/29) |
| 30 | [#30](https://github.com/mugiwaraluffy56/haedes/issues/30) |
| 31 | [#31](https://github.com/mugiwaraluffy56/haedes/issues/31) |
| 32 | [#32](https://github.com/mugiwaraluffy56/haedes/issues/32) |
| 33 | [#33](https://github.com/mugiwaraluffy56/haedes/issues/33) |
| 34 | [#34](https://github.com/mugiwaraluffy56/haedes/issues/34) |
| 35 | [#35](https://github.com/mugiwaraluffy56/haedes/issues/35) |
| 36 | [#36](https://github.com/mugiwaraluffy56/haedes/issues/36) |
| 37 | [#37](https://github.com/mugiwaraluffy56/haedes/issues/37) |
| 38 | [#38](https://github.com/mugiwaraluffy56/haedes/issues/38) |
| 39 | [#39](https://github.com/mugiwaraluffy56/haedes/issues/39) |
| 40 | [#40](https://github.com/mugiwaraluffy56/haedes/issues/40) |
| 41 | [#41](https://github.com/mugiwaraluffy56/haedes/issues/41) |
| 42 | [#42](https://github.com/mugiwaraluffy56/haedes/issues/42) |
| 43 | [#43](https://github.com/mugiwaraluffy56/haedes/issues/43) |
| 44 | [#44](https://github.com/mugiwaraluffy56/haedes/issues/44) |
| 45 | [#45](https://github.com/mugiwaraluffy56/haedes/issues/45) |

## Ticket board

### 01: Establish repository workspaces

**Plan coverage:** Task 1

**What to build:** Establish the monorepo foundation for TypeScript, Go, and Rust with explicit package and service boundaries.

**Blocked by:** None (can start immediately)

**Owner:** unassigned

**Status:** ready-for-agent

- [ ] Workspace manifests, package manager configuration, Go workspace, and Rust workspace are valid.
- [ ] Expected application, service, package, integration, infrastructure, test, and documentation boundaries are represented.
- [ ] No catch-all backend or unowned integration directory is introduced.

### 02: Add reproducible developer checks and ownership docs

**Plan coverage:** Task 1

**What to build:** A clean checkout has repeatable formatting, type checking, Go, Rust, generation, and repository validation commands with clear ownership rules.

**Blocked by:** 01: Establish repository workspaces

**Owner:** unassigned

**Status:** blocked

- [ ] Root check, test, typecheck, lint, and generation commands are documented and runnable.
- [ ] Generated contract drift fails deterministically.
- [ ] Ownership rules cover UI, SDK, MCP, control plane, runtime, and AWS/Terraform changes.

### 03: Define the canonical public API contract

**Plan coverage:** Task 2

**What to build:** Define the versioned `/v1` platform contract for sandbox lifecycle, commands, files, snapshots, authentication, pagination, idempotency, SSE, and typed errors.

**Blocked by:** 01: Establish repository workspaces

**Owner:** unassigned

**Status:** blocked

- [ ] Public schemas use sandbox, computer, command, file, workspace, and snapshot vocabulary rather than AWS internals.
- [ ] Routes, request/response bodies, status codes, headers, limits, IDs, timestamps, and error envelopes are specified.
- [ ] The contract supports the SDK, dashboard, MCP, future adapters, customer integrations, and internal tools.

### 04: Define runtime protocol and lifecycle contracts

**Plan coverage:** Task 2

**What to build:** Define the internal runtime protocol and lifecycle transition rules that connect the public platform contract to the Rust runtime.

**Blocked by:** 03: Define the canonical public API contract

**Owner:** unassigned

**Status:** blocked

- [ ] Runtime request/event schemas are versioned separately from the public API.
- [ ] All lifecycle states and legal transitions are represented.
- [ ] Fixtures cover command start, stdout, stderr, completion, file operations, snapshot export, and restore.

### 05: Generate and validate shared types

**Plan coverage:** Task 2

**What to build:** Generate shared TypeScript/API types and validate public, runtime, persisted, and MCP schemas from their canonical sources.

**Blocked by:** 03: Define the canonical public API contract; 04: Define runtime protocol and lifecycle contracts

**Owner:** unassigned

**Status:** blocked

- [ ] Generated TypeScript types and Go-facing models are reproducible.
- [ ] Contract validation checks lifecycle states, legal transitions, limits, opaque IDs, timestamps, and error requirements.
- [ ] MCP tool schemas reuse canonical types without creating duplicate domain models.

### 06: Bootstrap the Rust runtime service

**Plan coverage:** Task 3

**What to build:** Start a lightweight, authenticated Rust runtime inside a sandbox task with health endpoints and explicit configuration.

**Blocked by:** 01: Establish repository workspaces; 04: Define runtime protocol and lifecycle contracts

**Owner:** unassigned

**Status:** blocked

- [ ] The runtime starts as a non-root process and reports its version and workspace health.
- [ ] Internal runtime routes require a sandbox-scoped token; health remains separately available.
- [ ] Runtime failures use the agreed typed error vocabulary.

### 07: Enforce `/workspace` path isolation

**Plan coverage:** Task 3

**What to build:** Make every runtime filesystem path resolve safely beneath `/workspace`.

**Blocked by:** 06: Bootstrap the Rust runtime service

**Owner:** unassigned

**Status:** blocked

- [ ] Normal relative and absolute workspace paths resolve correctly.
- [ ] Traversal, encoded traversal, NUL bytes, unsafe symlinks, missing ancestors, and outside-root paths fail safely.
- [ ] Tests prove no runtime file operation can escape `/workspace`.

### 08: Implement runtime filesystem operations

**Plan coverage:** Task 3

**What to build:** The runtime can read, write, list, and delete workspace entries with bounded bodies and safe directory handling.

**Blocked by:** 07: Enforce `/workspace` path isolation

**Owner:** unassigned

**Status:** blocked

- [ ] File and directory operations work for normal, empty, UTF-8, and missing entries.
- [ ] Entry kinds, byte sizes, body limits, and deletion semantics are enforced.
- [ ] Filesystem tests cover symlink and limit behavior.

### 09: Implement runtime command execution

**Plan coverage:** Task 3

**What to build:** The Rust runtime runs bounded Linux commands with working directories, environment injection, process groups, exit status, and timeouts.

**Blocked by:** 06: Bootstrap the Rust runtime service

**Owner:** unassigned

**Status:** blocked

- [ ] Commands run in isolated process groups with bounded input and output.
- [ ] Success, non-zero exit, working directory, environment, shell input, and timeout behavior are tested.
- [ ] Timeout handling terminates descendants and returns a typed timeout result.

### 10: Add runtime event streaming

**Plan coverage:** Task 3

**What to build:** Runtime command output is published as ordered events that can be consumed and replayed after reconnect.

**Blocked by:** 09: Implement runtime command execution

**Owner:** unassigned

**Status:** blocked

- [ ] Started, stdout, stderr, completed, and failed events have monotonic sequence IDs.
- [ ] Output caps, stream closure, and terminal-event behavior are enforced.
- [ ] Tests cover mixed output, reconnect, output limits, and runtime failures.

### 11: Implement control-plane domain and ports

**Plan coverage:** Task 4

**What to build:** The Go control plane owns sandbox identity, lifecycle state, authorization boundaries, expiry calculation, metadata ports, compute ports, runtime ports, and snapshot coordination interfaces.

**Blocked by:** 04: Define runtime protocol and lifecycle contracts; 05: Generate and validate shared types

**Owner:** unassigned

**Status:** blocked

- [ ] Domain models and service methods cover create, get, list, destroy, execute, events, files, snapshot, and restore.
- [ ] Handlers and services depend on interfaces rather than AWS SDK calls.
- [ ] State mutation is centralized in the lifecycle state machine.

### 12: Add control-plane authentication and configuration

**Plan coverage:** Task 4

**What to build:** Protect the public control plane with owner-scoped API keys, safe configuration validation, request IDs, and secret-safe logging.

**Blocked by:** 03: Define the canonical public API contract; 11: Implement control-plane domain and ports

**Owner:** unassigned

**Status:** blocked

- [ ] Production startup rejects missing authentication/signing/AWS configuration.
- [ ] API keys are hashed at rest, compared safely, and mapped to owner principals.
- [ ] Every request receives a request ID and logs exclude credentials, environment values, and file contents.

### 13: Implement deterministic fake adapters

**Plan coverage:** Task 4

**What to build:** Supply in-memory metadata, compute, runtime, object-store, and event adapters for fast local tests.

**Blocked by:** 11: Implement control-plane domain and ports

**Owner:** unassigned

**Status:** blocked

- [ ] Fakes implement the same interfaces as production adapters.
- [ ] Fake start, stop, readiness, command, filesystem, snapshot, and restore events are observable.
- [ ] Tests cover provisioning failure, readiness failure, command routing, expiry, and missing sandboxes.

### 14: Expose sandbox lifecycle routes

**Plan coverage:** Task 5

**What to build:** Authenticated clients can create, inspect, list, and destroy sandboxes through the public API.

**Blocked by:** 12: Add control-plane authentication and configuration; 13: Implement deterministic fake adapters

**Owner:** unassigned

**Status:** blocked

- [ ] Routes return documented statuses, resource IDs, locations, lifecycle states, and typed errors.
- [ ] Limits, owner scope, idempotency, malformed input, and state conflicts are enforced.
- [ ] Handler tests cover create, get, list, destroy, and repeated destroy.

### 15: Expose command routes and public SSE

**Plan coverage:** Task 5

**What to build:** Authenticated clients can submit commands and subscribe to live ordered output through the public API.

**Blocked by:** 10: Add runtime event streaming; 14: Expose sandbox lifecycle routes

**Owner:** unassigned

**Status:** blocked

- [ ] Command submission returns `202` and a command ID.
- [ ] SSE forwards ordered events, flushes data, sends heartbeats, honors `Last-Event-ID`, and closes after terminal events.
- [ ] Public error envelopes cover state conflicts, timeouts, runtime failures, and missing commands.

### 16: Expose public filesystem routes

**Plan coverage:** Task 5

**What to build:** Authenticated clients can read, write, list, and delete workspace files through the public API.

**Blocked by:** 08: Implement runtime filesystem operations; 14: Expose sandbox lifecycle routes

**Owner:** unassigned

**Status:** blocked

- [ ] Routes enforce running state, owner scope, path safety, body limits, and bounded responses.
- [ ] The control plane delegates filesystem work to the runtime and does not implement its own filesystem logic.
- [ ] Handler tests cover success, missing files, traversal, authorization, and deletion.

### 17: Implement safe runtime snapshot archives

**Plan coverage:** Task 3 and Task 6

**What to build:** The runtime exports and restores compressed workspace archives with checksums and safe-entry validation.

**Blocked by:** 08: Implement runtime filesystem operations

**Owner:** unassigned

**Status:** blocked

- [ ] Export includes intended workspace content and calculates SHA-256 metadata.
- [ ] Restore rejects absolute paths, traversal, unsafe symlinks, and conflicting entries.
- [ ] Archive tests prove regular files and directories survive round trips.

### 18: Implement snapshot service and S3 metadata

**Plan coverage:** Task 6

**What to build:** Coordinate snapshot creation, checksum verification, encrypted S3 storage, metadata transitions, restore lookup, and retention.

**Blocked by:** 11: Implement control-plane domain and ports; 13: Implement deterministic fake adapters; 17: Implement safe runtime snapshot archives

**Owner:** unassigned

**Status:** blocked

- [ ] Snapshot metadata becomes available only after archive verification.
- [ ] Unknown snapshots, checksum failures, expired metadata, and retention deletion produce typed outcomes.
- [ ] Archive naming and object deletion are idempotent and private.

### 19: Implement lifecycle expiry and reconciliation workers

**Plan coverage:** Task 6

**What to build:** Periodic workers expire sandboxes, reconcile actual ECS task state, stop orphans, and make cleanup retries safe.

**Blocked by:** 11: Implement control-plane domain and ports; 13: Implement deterministic fake adapters; 21: Implement the ECS/Fargate provisioner

**Owner:** unassigned

**Status:** blocked

- [ ] Expired sandboxes are stopped once and eventually marked terminal.
- [ ] Orphaned tasks and stale starting sandboxes are detected and cleaned up.
- [ ] `--once` and bounded polling modes are testable without duplicate side effects.

### 20: Expose snapshot and restore routes

**Plan coverage:** Task 5

**What to build:** Authenticated clients can create snapshots and restore them into a fresh sandbox through the public API.

**Blocked by:** 14: Expose sandbox lifecycle routes; 17: Implement safe runtime snapshot archives; 18: Implement snapshot service and S3 metadata

**Owner:** unassigned

**Status:** blocked

- [ ] Snapshot creation is idempotent and exposes requested, creating, available, and failed states.
- [ ] Restore validates ownership, snapshot availability, target state, archive safety, and fresh-sandbox semantics.
- [ ] Route tests cover unknown snapshots, state conflicts, checksum failures, and successful restoration.

### 21: Implement the ECS/Fargate provisioner

**Plan coverage:** Task 9

**What to build:** Connect control-plane compute ports to ECS so one sandbox maps to one private Fargate task with a discoverable runtime endpoint.

**Blocked by:** 11: Implement control-plane domain and ports; 22: Build and harden the sandbox image; 24: Provision the AWS foundation with Terraform

**Owner:** unassigned

**Status:** blocked

- [ ] Start, describe, and stop operations use bounded deadlines and safe retries.
- [ ] Network attachment discovery returns the private runtime endpoint without persisting credentials in task metadata.
- [ ] Missing tasks, launch failures, transient throttling, and already-stopped tasks map safely.

### 22: Build and harden the sandbox image

**Plan coverage:** Task 7

**What to build:** Produce the pinned default development image used by local runtime tests and AWS Fargate tasks.

**Blocked by:** 06: Bootstrap the Rust runtime service; 01: Establish repository workspaces

**Owner:** unassigned

**Status:** blocked

- [ ] The image contains the planned development tools and starts the Rust runtime on the internal port.
- [ ] It runs non-root with writable workspace/temp directories and fails closed without a runtime token.
- [ ] It has no privileged mode, host mounts, Docker socket, or cloud credentials.

### 23: Add the local container workflow

**Plan coverage:** Task 7

**What to build:** Provide local compose and smoke-test workflows for the control plane, runtime, and fake dependencies without creating a local product mode.

**Blocked by:** 13: Implement deterministic fake adapters; 22: Build and harden the sandbox image

**Owner:** unassigned

**Status:** blocked

- [ ] Local containers use the same runtime contract as AWS tasks.
- [ ] Runtime access is restricted to the internal local network.
- [ ] Image smoke tests verify the required toolchain and health behavior.

### 24: Provision the AWS foundation with Terraform

**Plan coverage:** Task 8

**What to build:** Define the single-region development VPC, private subnets, routing, security groups, storage, metadata, logs, and ECR foundation.

**Blocked by:** 01: Establish repository workspaces; 03: Define the canonical public API contract

**Owner:** unassigned

**Status:** blocked

- [ ] Terraform formatting, validation, required tags, one-region constraints, and bounded resource inputs pass.
- [ ] S3 public access is blocked, encryption is enabled, DynamoDB recovery is configured, and logs have retention.
- [ ] Outputs provide the references required by the control plane and sandbox tasks.

### 25: Provision ECS services and least-privilege IAM

**Plan coverage:** Task 8

**What to build:** Define the public control-plane service and private per-sandbox task configuration with least-privilege roles.

**Blocked by:** 22: Build and harden the sandbox image; 24: Provision the AWS foundation with Terraform

**Owner:** unassigned

**Status:** blocked

- [ ] The control-plane role has only required ECS, S3, DynamoDB, and CloudWatch permissions.
- [ ] Sandbox tasks use `awsvpc`, no public IP, non-root runtime, private ingress, and no AWS credentials.
- [ ] Development and syntactically valid production compositions are separated.

### 26: Implement DynamoDB, S3, and retry adapters

**Plan coverage:** Task 9

**What to build:** Connect metadata and workspace storage ports to DynamoDB and S3 with conditional updates, streaming, checksums, encryption, and bounded retries.

**Blocked by:** 18: Implement snapshot service and S3 metadata; 24: Provision the AWS foundation with Terraform

**Owner:** unassigned

**Status:** blocked

- [ ] DynamoDB conditional state updates map to typed conflicts and missing-resource errors.
- [ ] S3 streaming verifies checksums, uses server-side encryption, and keeps objects private.
- [ ] Only transient throttling/network failures retry with bounded backoff and jitter.

### 27: Run AWS adapter and task smoke tests

**Plan coverage:** Task 9

**What to build:** Verify the real development deployment from API request through AWS resources and reconciliation.

**Blocked by:** 14: Expose sandbox lifecycle routes; 15: Expose command routes and public SSE; 16: Expose public filesystem routes; 20: Expose snapshot and restore routes; 21: Implement the ECS/Fargate provisioner; 26: Implement DynamoDB, S3, and retry adapters

**Owner:** unassigned

**Status:** blocked

- [ ] A real sandbox appears in ECS, writes metadata to DynamoDB, and emits CloudWatch logs.
- [ ] Commands and files work in the real task, and snapshots appear in encrypted S3.
- [ ] Destroy stops the task and reconciliation confirms no orphan remains.

### 28: Build the TypeScript SDK client core

**Plan coverage:** Task 10

**What to build:** Provide a configured TypeScript client and product-focused sandbox collection/handle over the canonical API.

**Blocked by:** 03: Define the canonical public API contract; 14: Expose sandbox lifecycle routes

**Owner:** unassigned

**Status:** blocked

- [ ] Create, get, list, and destroy are exposed without AWS implementation details.
- [ ] SDK configuration, injected fetch, request timeouts, request IDs, and typed errors are tested.
- [ ] The package has declarations, build output, and a product-focused README example.

### 29: Add SDK command and event support

**Plan coverage:** Task 10

**What to build:** SDK users can execute commands synchronously or consume typed live events with SSE reconnect behavior.

**Blocked by:** 15: Expose command routes and public SSE; 28: Build the TypeScript SDK client core

**Owner:** unassigned

**Status:** blocked

- [ ] `exec` returns command results and streaming yields ordered typed events.
- [ ] SSE parsing supports multiline data, reconnect, last-event IDs, completion, and failure.
- [ ] Tests cover headers, URLs, bodies, errors, timeouts, and stream ordering.

### 30: Add SDK filesystem and snapshot support

**Plan coverage:** Task 10

**What to build:** SDK users can read/write/list/delete files and create/restore snapshots through the sandbox abstraction.

**Blocked by:** 16: Expose public filesystem routes; 20: Expose snapshot and restore routes; 28: Build the TypeScript SDK client core

**Owner:** unassigned

**Status:** blocked

- [ ] File and snapshot methods use public product types and preserve typed errors.
- [ ] SDK examples cover create, work, snapshot, restore, and destroy.
- [ ] No method exposes ECS, Fargate, S3, IAM, task ARN, or security-group concepts.

### 31: Build the public haedes landing page

**Plan coverage:** Task 11

**What to build:** Explain haedes as the execution layer that gives agents computers, including the brain/computer distinction, AWS lifecycle, and SDK abstraction.

**Blocked by:** 03: Define the canonical public API contract; 28: Build the TypeScript SDK client core

**Owner:** unassigned

**Status:** blocked

- [ ] The page leads with “Give your AI agent a computer on AWS.”
- [ ] It explains MCP, SDK, HTTP API, Go control plane, Rust runtime, and AWS behavior without claiming unsupported vendor integration.
- [ ] It links to the dashboard and uses a product-level SDK example.

### 32: Build the dashboard lifecycle surface

**Plan coverage:** Task 11

**What to build:** Humans can see sandbox lists, lifecycle states, expiry, resource allocation, AWS status, and errors using the real API.

**Blocked by:** 14: Expose sandbox lifecycle routes; 28: Build the TypeScript SDK client core

**Owner:** unassigned

**Status:** blocked

- [ ] Loading, empty, provisioning, running, snapshotting, stopping, failed, stopped, and destroyed states are visible.
- [ ] Non-terminal sandboxes poll; terminal sandboxes do not poll unnecessarily.
- [ ] Errors display request IDs and no screen silently replaces real API data with mocks.

### 33: Add dashboard terminal, timeline, and snapshot controls

**Plan coverage:** Task 11

**What to build:** Make agent activity legible with live terminal output, execution history, AWS status, snapshots, restore, and destroy controls.

**Blocked by:** 15: Expose command routes and public SSE; 20: Expose snapshot and restore routes; 32: Build the dashboard lifecycle surface

**Owner:** unassigned

**Status:** blocked

- [ ] Live stdout/stderr, command status, duration, execution timeline, and reconnect behavior are visible.
- [ ] Snapshot creation, restore into a new sandbox, AWS resource status, and destruction are observable.
- [ ] The dashboard remains a human control surface while the agent is the primary actor.

### 34: Implement GitHub repository integration

**Plan coverage:** Task 12

**What to build:** Validate repository references and prepare safe clone requests without exposing credentials to commands, logs, agents, or sandbox tasks.

**Blocked by:** 05: Generate and validate shared types; 11: Implement control-plane domain and ports

**Owner:** unassigned

**Status:** blocked

- [ ] HTTPS GitHub URLs and owner/repository shorthand normalize correctly.
- [ ] Invalid hosts, malformed repositories, embedded credentials, and unsafe inputs return typed errors.
- [ ] Credentials come from an injected boundary and are redacted from command/event output.

### 35: Build provider-neutral agent session examples and fixture

**Plan coverage:** Task 12

**What to build:** Provide a deterministic fixture repository and provider-neutral examples that exercise create, command, file, snapshot, restore, and destroy operations without implementing a model loop.

**Blocked by:** 28: Build the TypeScript SDK client core; 30: Add SDK filesystem and snapshot support; 34: Implement GitHub repository integration

**Owner:** unassigned

**Status:** blocked

- [ ] The fixture begins with a failing authentication test and has a deterministic passing fix.
- [ ] Basic and coding-agent examples use the sandbox abstraction and return useful identities/results.
- [ ] The agent session facade remains a host contract, not a generic agent framework.

### 36: Define the MCP tool contract and server bootstrap

**Plan coverage:** Task 2 and Task 12

**What to build:** Define the single MCP integration boundary, tool schemas, structured results, and server configuration over the existing platform contract.

**Blocked by:** 05: Generate and validate shared types; 28: Build the TypeScript SDK client core

**Owner:** unassigned

**Status:** blocked

- [ ] Tools cover lifecycle, command, streaming, filesystem, snapshot, restore, and destroy operations.
- [ ] Schemas contain bounded product inputs and sandbox/command/snapshot identities, not AWS internals.
- [ ] Server configuration holds the public API credential outside tool input and model-visible output.

### 37: Implement MCP tool behavior and error mapping

**Plan coverage:** Task 12

**What to build:** Agents can use every MCP tool through the SDK or shared HTTP client with structured results and stable platform-error mapping.

**Blocked by:** 15: Expose command routes and public SSE; 16: Expose public filesystem routes; 20: Expose snapshot and restore routes; 29: Add SDK command and event support; 30: Add SDK filesystem and snapshot support; 36: Define the MCP tool contract and server bootstrap

**Owner:** unassigned

**Status:** blocked

- [ ] Create, get, list, exec, stream, read, write, list files, delete file, snapshot, restore, and destroy work.
- [ ] Authentication, ownership, state, timeout, runtime, and nonexistent-sandbox failures reach the agent clearly.
- [ ] MCP has no direct AWS dependency, independent persistence, or second lifecycle state machine.

### 38: Test the MCP adapter against a fake control plane

**Plan coverage:** Task 12 and Task 13

**What to build:** Validate MCP schemas, tool inputs/outputs, fake-platform behavior, authentication, error propagation, and sandbox identity handling before connecting a real coding agent.

**Blocked by:** 13: Implement deterministic fake adapters; 37: Implement MCP tool behavior and error mapping

**Owner:** unassigned

**Status:** blocked

- [ ] Schema validity and undocumented-field rejection are tested.
- [ ] Create, exec, stream, read, write, list, delete, snapshot, restore, and destroy are tested through the fake control plane.
- [ ] Tests prove AWS credentials, runtime tokens, cross-owner IDs, and lifecycle-only fields cannot enter through tools.

### 39: Verify the real coding-agent integration

**Plan coverage:** Task 12 and Task 13

**What to build:** Connect one existing coding-agent client to MCP and verify the complete bug-fix workflow in the real environment.

**Blocked by:** 22: Build and harden the sandbox image; 27: Run AWS adapter and task smoke tests; 33: Add dashboard terminal, timeline, and snapshot controls; 35: Build provider-neutral agent session examples and fixture; 38: Test the MCP adapter against a fake control plane

**Owner:** unassigned

**Status:** blocked

- [ ] Run and record the compatibility spike, exact client/version, MCP configuration, and invocation.
- [ ] The agent creates a sandbox, clones the fixture, installs dependencies, observes failure, edits files, reruns successfully, snapshots, destroys, restores, retests, and destroys again.
- [ ] No Claude Code, Codex, or other vendor capability is claimed unless verified in this environment.

### 40: Add integration and end-to-end journey tests

**Plan coverage:** Task 13

**What to build:** Verify the complete control-plane/runtime, snapshot, SDK, MCP, and dashboard journeys using local fakes and opt-in AWS tests.

**Blocked by:** 15: Expose command routes and public SSE; 16: Expose public filesystem routes; 20: Expose snapshot and restore routes; 29: Add SDK command and event support; 30: Add SDK filesystem and snapshot support; 33: Add dashboard terminal, timeline, and snapshot controls; 38: Test the MCP adapter against a fake control plane

**Owner:** unassigned

**Status:** blocked

- [ ] Control-plane/runtime and snapshot round-trip tests pass with fake AWS adapters and a real runtime container.
- [ ] SDK, MCP, and dashboard journeys cover create, work, snapshot, restore, and destroy.
- [ ] Opt-in AWS smoke tests are isolated from production and verify the real resource path.

### 41: Add security and abuse-case tests

**Plan coverage:** Task 13

**What to build:** Prove that authentication, task-token, filesystem, archive, resource, and process boundaries fail safely.

**Blocked by:** 07: Enforce `/workspace` path isolation; 10: Add runtime event streaming; 12: Add control-plane authentication and configuration; 17: Implement safe runtime snapshot archives; 19: Implement lifecycle expiry and reconciliation workers; 22: Build and harden the sandbox image; 38: Test the MCP adapter against a fake control plane

**Owner:** unassigned

**Status:** blocked

- [ ] Unauthenticated access, cross-owner IDs, invalid runtime tokens, traversal, symlink escapes, and unsafe archives fail.
- [ ] Oversized bodies/output, command timeouts, and descendant processes are bounded and terminated.
- [ ] Tests prove sandbox tasks have no AWS credentials, host mounts, Docker socket, or public snapshot access.

### 42: Add load scenarios and operational metrics

**Plan coverage:** Task 13 and Observability

**What to build:** Measure bounded concurrent command streams and sandbox lifecycle operations while emitting the metrics needed to diagnose the demo environment.

**Blocked by:** 19: Implement lifecycle expiry and reconciliation workers; 27: Run AWS adapter and task smoke tests; 40: Add integration and end-to-end journey tests

**Owner:** unassigned

**Status:** blocked

- [ ] Load scenarios cover concurrent command streams and repeated create/destroy requests.
- [ ] Metrics cover create/readiness/destroy/orphans, command duration/timeouts/event delay, snapshots, and runtime failures.
- [ ] Results are recorded as local or development measurements, not unsupported production claims.

### 43: Add CI and release quality gates

**Plan coverage:** Task 14

**What to build:** Make pull requests and clean checkouts validate contracts, language tooling, containers, Terraform, dependencies, secrets, and security tests.

**Blocked by:** 02: Add reproducible developer checks and ownership docs; 23: Add the local container workflow; 25: Provision ECS services and least-privilege IAM; 40: Add integration and end-to-end journey tests; 41: Add security and abuse-case tests; 42: Add load scenarios and operational metrics

**Owner:** unassigned

**Status:** blocked

- [ ] CI runs formatting, type checking, contract generation, Go tests/vet, Rust checks/tests, container checks, Terraform validation, and security checks.
- [ ] Secret scanning, dependency audits, IaC checks, and contract drift fail safely.
- [ ] A clean checkout can reproduce CI through local commands.

### 44: Write architecture, security, AWS, and demo runbooks

**Plan coverage:** Task 14

**What to build:** Document the system, data flow, runtime limits, threat model, AWS deployment, console verification, and coding-agent demo procedure.

**Blocked by:** 21: Implement the ECS/Fargate provisioner; 24: Provision the AWS foundation with Terraform; 25: Provision ECS services and least-privilege IAM; 26: Implement DynamoDB, S3, and retry adapters; 27: Run AWS adapter and task smoke tests; 33: Add dashboard terminal, timeline, and snapshot controls; 39: Verify the real coding-agent integration

**Owner:** unassigned

**Status:** blocked

- [ ] Runbooks explain what AWS resources were created, how to inspect ECS, CloudWatch, S3, DynamoDB, and orphan tasks, and how to clean up.
- [ ] Architecture and security docs state the ownership boundaries, no-credential model, `/workspace` isolation, lifecycle, and snapshot behavior.
- [ ] The demo script begins in the verified coding agent; curl is retained for debugging and infrastructure verification.

### 45: Perform final release rehearsal and scope audit

**Plan coverage:** Task 15

**What to build:** Rehearse the complete haedes flow from a clean environment and produce the final submission-ready scope, evidence, and documentation.

**Blocked by:** 39: Verify the real coding-agent integration; 40: Add integration and end-to-end journey tests; 41: Add security and abuse-case tests; 43: Add CI and release quality gates; 44: Write architecture, security, AWS, and demo runbooks

**Owner:** unassigned

**Status:** blocked

- [ ] The agent creates, uses, snapshots, destroys, restores, retests, and destroys again without manual state repair.
- [ ] The team verifies ECS lifecycle, CloudWatch logs, S3 snapshot, DynamoDB metadata, stopped tasks, no public objects, and no sandbox credentials.
- [ ] The scope audit maps every PRD must-build item to a completed ticket and confirms deferred features stayed out.
- [ ] The final video stays within three minutes and shows the agent, MCP, dashboard, and required AWS evidence.

## Plan-to-ticket traceability

| Plan task | Covered by tickets |
| --- | --- |
| Task 1: Repository tooling and ownership | 01–02 |
| Task 2: Canonical contracts and lifecycle rules | 03–05, 36 |
| Task 3: Rust sandbox runtime core | 06–10, 17 |
| Task 4: Go control-plane domain and ports | 11–13 |
| Task 5: Public Go API and SSE | 14–16, 20 |
| Task 6: Lifecycle and snapshot worker boundaries | 17–19 |
| Task 7: Images and local containers | 22–23 |
| Task 8: AWS development environment with Terraform | 24–25 |
| Task 9: AWS adapters and reconciliation | 21, 26–27 |
| Task 10: TypeScript SDK | 28–30 |
| Task 11: Web and dashboard | 31–33 |
| Task 12: GitHub and coding-agent integrations | 34–39 |
| Task 13: Integration, e2e, security, and load tests | 38, 40–42 |
| Task 14: CI, release checks, and operational docs | 43–44 |
| Task 15: Release rehearsal and scope audit | 45 |

## Explicitly deferred

Multiple agents, multiple MCP implementations, provider-specific plugins, Python SDKs, browser automation, GPUs, Kubernetes, custom sandbox images beyond the single default image, billing, organizations, multiple regions, marketplace features, advanced networking controls, and autoscaling remain outside the current plan.
