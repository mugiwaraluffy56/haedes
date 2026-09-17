# Control plane

The Go control plane owns the canonical `/v1` API, sandbox identity, lifecycle state, authorization, and AWS orchestration. It depends on interfaces and adapters rather than exposing provider-specific details to handlers.
