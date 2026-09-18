# Coding-agent sandbox example

This directory contains the deterministic SDK journey used to exercise the
host boundary. The opt-in real-client journey is launched through the MCP
server by [`scripts/run-coding-agent-journey.mjs`](../../../scripts/run-coding-agent-journey.mjs).
It creates a sandbox for a GitHub repository, installs dependencies, runs the
failing fixture test, reads and edits the authentication source, reruns the
test, saves a snapshot, destroys the original sandbox, restores the snapshot
into a new sandbox, reruns the tests, and destroys the restored sandbox.

The deterministic edit is only a compatibility example; there is no model
planner or vendor-specific agent loop here. Build the MCP package first, then
configure `HAEDES_API_URL`, `HAEDES_API_KEY`, and `HAEDES_REPOSITORY_URL` and
run:

```sh
pnpm --filter @haedes/mcp build
node scripts/run-coding-agent-journey.mjs
```

Set `HAEDES_REPOSITORY_PATH` or `HAEDES_REPOSITORY_REF` when the repository
uses different values. The recorded compatibility status is in
[`docs/agent-compatibility.md`](../../../docs/agent-compatibility.md). Use
`--dry-run` to inspect the exact client version, MCP configuration, and
non-interactive invocation without contacting the agent service. The script exits unless the client reports
`journeyStatus: "passed"`; do not claim a client is verified from a dry run.
