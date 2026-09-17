# Sandbox lifecycle contract

The control plane owns sandbox lifecycle state. The runtime only reports health and execution results; it cannot transition its own public lifecycle state.

## States

| State | Meaning |
| --- | --- |
| requested | The create request was accepted and the sandbox resource was persisted. |
| provisioning | The control plane is creating the temporary computer. |
| starting | The computer exists and the runtime is undergoing its health check. |
| running | The runtime is healthy and accepts commands and filesystem operations. |
| snapshotting | A workspace archive is being created. Mutating operations are paused. |
| stopping | Destruction or expiration has begun and new work is rejected. |
| stopped | The computer has stopped but its metadata is retained. |
| failed | Provisioning, startup, runtime, or cleanup failed. The failure reason is retained. |
| destroyed | The terminal state. The computer and its access path are gone. |

## Legal transitions

| From | To | Trigger and required side effect |
| --- | --- | --- |
| requested | provisioning | Persist the resource before starting provisioning work. |
| requested | failed | Reject invalid configuration or record a provisioning failure. |
| provisioning | starting | Persist the task endpoint after the compute provider reports it. |
| provisioning | failed | Record a compute launch or provisioning failure. |
| starting | running | Runtime health check succeeds; update lastActivityAt. |
| starting | failed | Startup deadline expires or health check fails. |
| running | snapshotting | Snapshot request is accepted; reject new commands while active. |
| running | stopping | Destroy or expiration is accepted; reject new work. |
| running | failed | An unrecoverable runtime or reconciliation error occurs. |
| snapshotting | running | Archive is stored successfully; preserve workspace on failure. |
| snapshotting | failed | Snapshot creation or upload fails; retain the prior workspace. |
| stopping | stopped | The compute task stops and reconciliation confirms it. |
| stopping | destroyed | Cleanup confirms the task and access path are absent. |
| stopping | failed | The stop deadline expires; retain the cleanup failure. |
| stopped | destroyed | Explicit destroy or retention cleanup completes. |
| failed | destroyed | Best-effort cleanup completes; retain the original failure. |

Every transition not listed above is invalid. In particular, a destroyed sandbox cannot be restarted or mutated, and a stopped sandbox cannot accept commands. Transition checks are centralized in the control plane so retries cannot create divergent state.

## Retry and idempotency rules

- Repeating destroy for a destroyed sandbox is a successful no-op.
- Repeating create or snapshot is safe only when the same owner supplies the same Idempotency-Key.
- A failed snapshot leaves the current workspace and prior snapshot metadata unchanged.
- A failed cleanup remains failed until an explicit cleanup request reaches destroyed.
- Runtime events are ordered by a monotonic sequence per command. Reconnecting clients resume from Last-Event-ID.
