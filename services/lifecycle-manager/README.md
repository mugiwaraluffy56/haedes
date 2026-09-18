# Lifecycle manager

The Go lifecycle manager owns expiration and reconciliation work for named sandboxes. It does not become a second control plane.

The `internal/worker` package exposes the `LifecycleWorker` behavior: bounded
expiration, stale-start cleanup, ECS task reconciliation, orphan cleanup, and
atomic stop-request claims. A stop request is persisted before the compute
operation is considered complete, so retries can safely distinguish an
already-issued stop from a failed stop. The worker never invents lifecycle
transitions; the repository applies the control-plane transition table.

The worker binary supports `--once` for CI and bounded polling flags for
deployment. It intentionally refuses to start until concrete repository and
compute adapters are wired, rather than silently falling back to in-memory
state or fake ECS behavior.
