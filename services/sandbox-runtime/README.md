# Sandbox runtime

The Rust runtime runs inside one sandbox task and owns child processes, command output, filesystem operations, and `/workspace`. It must not contain lifecycle, dashboard, or AWS application logic.

## Bootstrap

The process requires `HAEDES_RUNTIME_TOKEN` and `HAEDES_SANDBOX_ID`. Optional settings are `HAEDES_RUNTIME_BIND` (default `0.0.0.0:8080`), `HAEDES_RUNTIME_VERSION`, and `HAEDES_RUNTIME_WORKSPACE` (default `/workspace`). Startup rejects root execution, missing workspace directories, invalid sandbox IDs, relative workspace paths, and empty tokens.

`GET /healthz` is unauthenticated and reports the runtime version, sandbox ID, and workspace availability. Every `/v1/*` route is protected by `Authorization: Bearer <sandbox-token>` and returns the typed `runtime_unauthorized` error envelope when the token is missing or invalid. `/v1/status` is a bootstrap status route used by the runtime tests; command, filesystem, streaming, and snapshot routes are added by the subsequent runtime tasks.
