# Control plane

The Go control plane owns the canonical `/v1` API, sandbox identity, lifecycle state, authorization, and AWS orchestration. It depends on interfaces and adapters rather than exposing provider-specific details to handlers.

The `internal/sandbox` package contains the provider-neutral domain model, centralized lifecycle transition validation, orchestration service, and ports for persistence, compute, runtime, snapshots, authentication, IDs, and time. AWS adapters can implement those ports without changing the domain service.

The `internal/api` package exposes the authenticated sandbox lifecycle routes. It owns HTTP validation, owner scoping, idempotent create handling, request IDs, and conversion to the generated public contract types.

The control-plane binary exposes an internal `/metrics` endpoint with
Prometheus text for sandbox create/readiness/destroy/orphan activity, command
duration/timeouts/event delivery, snapshot bytes/failures/restores, and runtime
failures. Values are diagnostic local/development measurements, not production
capacity claims.

The `internal/aws/ecs` package implements the `sandbox.ComputeProvisioner` port
with the AWS ECS SDK. It launches one Fargate task into private subnets with
public IP assignment disabled, injects only the sandbox-scoped runtime token
and ID, waits for `RUNNING`, discovers the ENI private IPv4 address, and
returns the internal runtime endpoint. Calls use bounded contexts and retry
only throttling or server faults. Missing tasks and already-stopped tasks are
safe cleanup outcomes; AWS credentials are rejected from task environment
overrides.
