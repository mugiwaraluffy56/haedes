# Coding-agent demo runbook

The demo starts in the verified coding-agent client. `curl` is a debugging and
infrastructure-verification fallback, not the primary product experience.

## Preconditions

- A deployed control plane is healthy at the ALB URL.
- The sandbox and control-plane images are pushed to ECR and the ECS service is
  stable.
- Claude Code compatibility has been verified for the client version recorded
  in [agent-compatibility.md](agent-compatibility.md). A dry run alone is not
  compatibility verification.
- The repository URL points to a cloneable fixture with `package.json`,
  `src/auth.ts`, and the intentionally failing authentication test. The local
  source fixture is under `internal/fixtures/auth-bug-repository` and must be
  published to a permitted Git provider before an AWS run.
- The operator has a short-lived development API key and an AWS smoke cleanup
  owner.

## Agent-first run

Build the MCP server, configure only the server-side connection values, and
launch the harness:

```sh
pnpm --filter @haedes/mcp build

export HAEDES_API_URL="http://<control-plane-alb>"
export HAEDES_API_KEY="<short-lived-development-key>"
export HAEDES_REPOSITORY_URL="https://github.com/<owner>/<fixture>.git"
export HAEDES_REPOSITORY_PATH=/workspace/fixture

node scripts/run-coding-agent-journey.mjs
```

The harness writes a temporary stdio MCP configuration and invokes Claude Code
with strict MCP configuration, no session persistence, and no local tool
access. The prompt requires the agent to:

1. Create a sandbox and clone the fixture.
2. Install dependencies and observe the expected failing test.
3. Read and minimally edit `src/auth.ts` through MCP file tools.
4. Rerun the test successfully.
5. Snapshot the workspace, destroy the original sandbox, restore into a new
   sandbox, and rerun the test.
6. Destroy the restored sandbox and return a JSON result with both IDs,
   snapshot ID, observed failure/success, and cleanup status.

The command is successful only when the agent reports
`journeyStatus: "passed"`. Save the redacted output, timestamp, client
version, commit/image digests, sandbox IDs, and snapshot ID as demo evidence.
Never save API keys, runtime tokens, repository credentials, or full task
metadata.

## Compatibility preflight

Before a live run, verify the exact client and invocation without contacting
the agent service:

```sh
node scripts/run-coding-agent-journey.mjs --dry-run
```

The output must identify the client version, temporary MCP config, Node-launched
`integrations/mcp/dist/server.js`, and the non-interactive invocation. Record
the output after redacting the temporary path if publishing evidence.

## Debugging fallback

If the agent fails, preserve the agent output and correlate its sandbox ID with
the [AWS operations runbook](aws/operations.md). Use the public API with `curl`
only to inspect or isolate a failing boundary:

```sh
curl --fail "$HAEDES_API_URL/healthz"
curl --fail -H "Authorization: Bearer $HAEDES_API_KEY" \
  "$HAEDES_API_URL/v1/sandboxes?limit=20"
```

For the low-level lifecycle check, use the opt-in
[`scripts/smoke-aws.sh`](../scripts/smoke-aws.sh), which covers health, create,
command execution, file write/read, snapshot, and cleanup. It does not prove
coding-agent compatibility.

## Stop conditions

Stop and clean up if the agent reports a failed journey, a sandbox remains
running after the final response, an ECS task has no matching record, or a
snapshot checksum does not match. Do not retry blindly: inspect CloudWatch,
DynamoDB, ECS, and S3 using the operations runbook, then destroy any confirmed
orphan task through the documented procedure.
