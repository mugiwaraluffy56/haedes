# Public API contract

The canonical public contract is [`internal/contracts/api.openapi.yaml`](../../internal/contracts/api.openapi.yaml). It is versioned under `/v1` and is the only lifecycle and resource contract that SDK, dashboard, MCP, future adapters, customer integrations, and internal tools should consume.

## Contract rules

- Public vocabulary is sandbox, command, file, workspace, and snapshot. AWS identifiers never appear in public schemas.
- API keys use `Authorization: Bearer <key>`. The key is never accepted as a tool argument or forwarded to a sandbox task.
- Every response includes an `X-Request-ID` correlation header. Errors use the `{ "error": { "code", "message", "requestId", "details" } }` envelope.
- Resource identifiers are opaque and prefixed with `sbx_`, `cmd_`, `snp_`, or `req_`.
- Timestamps are RFC 3339 strings with an explicit timezone.
- List endpoints use opaque cursors and a bounded `limit` from 1 through 100.
- Sandbox and snapshot creation accept a required `Idempotency-Key` for safe retries.
- Commands are asynchronous and return `202` with a command ID. Output is ordered, replayable Server-Sent Events using `Last-Event-ID`.
- Command input is bounded to 16 KiB, environment maps to 32 entries, file bodies to 10 MiB, and command timeouts to 900 seconds.
- File paths must remain beneath `/workspace`; the runtime owns the final traversal and symlink checks.

Non-success responses use typed HTTP errors. In particular, `401` means authentication failed, `404` means the resource is absent or not visible to the caller, `409` means a lifecycle or idempotency conflict, `408` means a command exceeded its timeout, and `413` means a configured request limit was exceeded.

Run `make generate` to verify the generated TypeScript and Go consumers match this source. Run `pnpm validate:contracts` to validate the public, runtime, persisted, and MCP boundaries.
