# Sandbox runtime

The Rust runtime runs inside one sandbox task and owns child processes, command output, filesystem operations, and `/workspace`. It must not contain lifecycle, dashboard, or AWS application logic.

## Bootstrap

The process requires `HAEDES_RUNTIME_TOKEN` and `HAEDES_SANDBOX_ID`. Optional settings are `HAEDES_RUNTIME_BIND` (default `0.0.0.0:8080`), `HAEDES_RUNTIME_VERSION`, and `HAEDES_RUNTIME_WORKSPACE` (default `/workspace`). Startup rejects root execution, missing workspace directories, invalid sandbox IDs, relative workspace paths, and empty tokens.

`GET /healthz` is unauthenticated and reports the runtime version, sandbox ID, and workspace availability. Every `/v1/*` route is protected by `Authorization: Bearer <sandbox-token>` and returns the typed `runtime_unauthorized` error envelope when the token is missing or invalid. `/v1/status` is a bootstrap status route used by the runtime tests; command, filesystem, streaming, and snapshot routes are added by the subsequent runtime tasks.

Filesystem paths are resolved through `PathGuard`. Relative paths and `/workspace/...` paths map beneath the configured workspace root. Traversal, encoded path escapes, NUL bytes, missing ancestors, outside-root paths, and symlinks that resolve outside the workspace are rejected with typed path errors.

`FileService` provides bounded read and write operations, deterministic directory listings, and file or empty-directory deletion. It reports regular files, directories, and symlinks separately. Callers can supply a runtime-specific limit; the default limit is 10 MiB.

`CommandRunner` executes bounded commands through `sh -lc` with a workspace working directory, an explicit environment, isolated process groups, separate stdout and stderr limits, and a timeout capped at 15 minutes. A timeout returns a result with `timed_out: true`; output overflow returns the typed `command_output_limit` error.
