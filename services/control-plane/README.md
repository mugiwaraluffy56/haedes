# Control plane

The Go control plane owns the canonical `/v1` API, sandbox identity, lifecycle state, authorization, and AWS orchestration. It depends on interfaces and adapters rather than exposing provider-specific details to handlers.

The `internal/sandbox` package contains the provider-neutral domain model, centralized lifecycle transition validation, orchestration service, and ports for persistence, compute, runtime, snapshots, authentication, IDs, and time. AWS adapters can implement those ports without changing the domain service.

The `internal/api` package exposes the authenticated sandbox lifecycle routes. It owns HTTP validation, owner scoping, idempotent create handling, request IDs, and conversion to the generated public contract types.
