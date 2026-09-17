# Internal runtime protocol

The runtime protocol is defined by the runtime-v1.schema.json file in packages/protocol. It is separate from the public OpenAPI contract because it is an authenticated, private protocol between the Go control plane and the Rust runtime inside a sandbox task.

## Boundary

The runtime exposes only these internal routes:

~~~
GET    /healthz
POST   /v1/exec
GET    /v1/exec/{commandId}/events
GET    /v1/files?path=/workspace
GET    /v1/files/content?path=/workspace/file
PUT    /v1/files/content?path=/workspace/file
DELETE /v1/files/content?path=/workspace/file
POST   /v1/snapshot/export
POST   /v1/snapshot/restore
~~~

/healthz is available for task health checks. Every /v1/* route requires a short-lived, sandbox-scoped runtime token. Public API keys, user identity, dashboard requests, AWS credentials, and provider identifiers never cross this boundary.

## Message rules

- Every JSON message has version: "runtime.v1" and an explicit type.
- Runtime request IDs use the req_ prefix; command and snapshot IDs use cmd_ and snp_.
- Every timestamp is RFC 3339 with an explicit timezone.
- Commands are bounded to 16 KiB, environment maps to 32 entries, file bodies to 10 MiB, and timeouts to 900 seconds.
- Workspace paths are rooted at /workspace; traversal and symlink escape checks remain mandatory runtime behavior.
- Command events contain monotonic sequence values. started, stdout, stderr, and exactly one terminal completed or failed event form the command stream.
- File contents use base64 in JSON messages so binary data is not silently coerced into text.
- Snapshot export and restore report metadata only. The runtime does not receive public S3 URLs or AWS credentials.

Fixtures under packages/protocol/src/fixtures are small, stable examples for schema and adapter tests. They are not production data.
