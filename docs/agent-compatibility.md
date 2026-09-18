# Coding-agent MCP compatibility

The repository provides an opt-in compatibility harness for one existing
coding-agent client. It uses Claude Code through its stdio MCP configuration;
the MCP server remains provider-neutral and owns no model loop.

## Current record

- Client: Claude Code `2.1.145` was present in the development environment.
- Transport: stdio, launched with Node from `integrations/mcp/dist/server.js`.
- Configuration: the harness writes a temporary configuration containing the
  `HAEDES_API_URL` and `HAEDES_API_KEY` environment variables for the MCP
  subprocess. The API key is never placed in the prompt or tool arguments.
- Invocation: `node scripts/run-coding-agent-journey.mjs`.
- Status: the client was dry-run verified for version/configuration/invocation
  construction. The live sandbox journey remains unverified until a deployed
  HAEDES API, API key, and cloneable repository are supplied.

## Run the verification

```sh
pnpm --filter @haedes/mcp build
export HAEDES_API_URL=https://<control-plane>
export HAEDES_API_KEY=<api-key>
export HAEDES_REPOSITORY_URL=https://github.com/<owner>/<repository>.git
node scripts/run-coding-agent-journey.mjs
```

The agent must report `journeyStatus: "passed"` after creating a sandbox,
installing dependencies, observing the fixture failure, editing the fixture,
passing the test, snapshotting, destroying, restoring, retesting, and
destroying the restored sandbox. A dry run is available with
`node scripts/run-coding-agent-journey.mjs --dry-run`, but it must not be
described as client compatibility verification.
