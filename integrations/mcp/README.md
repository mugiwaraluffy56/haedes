# haedes MCP boundary

The MCP integration is a thin agent-facing adapter over the public sandbox API.
It defines one bounded tool for each product operation:

`sandbox_create`, `sandbox_get`, `sandbox_list`, `sandbox_exec`,
`sandbox_exec_stream`, `sandbox_read_file`, `sandbox_write_file`,
`sandbox_list_files`, `sandbox_delete_file`, `sandbox_snapshot`,
`sandbox_restore`, and `sandbox_destroy`.

The contract is versioned in
[`src/tool-contract.schema.json`](src/tool-contract.schema.json). Inputs use
product identities and workspace paths, enforce bounded command/file/resource
values, reject undocumented fields, and never accept API keys, runtime tokens,
or infrastructure identifiers. Results are structured envelopes that preserve
sandbox, command, and snapshot identities.

Configure the server outside tool input with:

```sh
export HAEDES_API_URL=http://localhost:8080
export HAEDES_API_KEY=development-key
node dist/server.js
```

The API key stays in the server-side `SandboxClient`; it is not included in
tool definitions, arguments, results, or sandbox requests. The default
handlers delegate every operation to `@haedes/sdk`, including bounded command
event collection and workspace file operations. Platform errors are returned
as structured MCP tool errors with the public API code, HTTP status, request
ID, and safe API details. This package owns no lifecycle state, persistence,
AWS access, or model/provider loop.
