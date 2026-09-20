# Final release rehearsal

This is the release gate for the hackathon slice. It defines one reproducible
agent path, the AWS evidence that must be captured, the failure cases that must
be exercised, and the claims that are safe to make in the final submission.

The rehearsal is successful only when the live run completes with
`journeyStatus: "passed"`, both sandbox tasks are destroyed, and the redacted
evidence packet contains the matching API, ECS, CloudWatch, S3, and DynamoDB
identifiers. A local dry run or a fake-provider test is useful preflight; it is
not evidence of a live coding-agent or AWS rehearsal.

## Gate 0: clean checkout

Run from a clean checkout on the release commit. Keep credentials outside the
repository and use a temporary evidence directory.

```sh
git clone <repository-url> haedes-release
cd haedes-release
git switch main
git pull --ff-only origin main
./scripts/check-repo.sh
node scripts/run-coding-agent-journey.mjs --dry-run
make check
```

`make check` is the repository gate. If a local dependency, Docker daemon,
Terraform installation, or AWS account is unavailable, record the exact
command and environment limitation; do not replace a missing live check with a
success claim.

## Gate 1: live preflight

Before creating a sandbox, record these values in a redacted evidence manifest:

| Field | Required value |
| --- | --- |
| Release commit | Immutable Git commit SHA |
| Control-plane image | ECR repository and digest |
| Sandbox image | ECR repository and digest |
| Agent client | Exact client and version from the compatibility spike |
| AWS account and region | Account ID suffix and configured region |
| Operator | Cleanup owner and UTC start time |

Use a short-lived API key. Do not place it, runtime tokens, repository
credentials, full ECS task metadata, or unredacted logs in the evidence packet.
The exact client invocation and MCP configuration must pass the compatibility
preflight in [the coding-agent demo runbook](coding-agent-demo.md).

## Gate 2: one vertical slice

Run the agent-first path once, then repeat the client-level lifecycle through
the SDK. The following evidence IDs must refer to the same run where possible.

| Step | Expected result | Evidence to retain |
| --- | --- | --- |
| Agent calls `sandbox_create` | `requested → provisioning → starting → running` | sandbox ID, request ID, dashboard capture |
| Clone fixture and install dependencies | Work happens inside `/workspace/fixture` | command IDs and redacted output |
| Run the fixture test | Intentional failure is observed | failed command ID and exit code |
| Read and edit the fixture | Only the intended file changes | file operation IDs and diff summary |
| Rerun the test | Test passes with live streamed output | passing command ID |
| Create snapshot | S3 object exists and checksum is recorded | snapshot ID, object key, checksum |
| Destroy original sandbox | ECS task reaches stopped | task ARN suffix and stopped timestamp |
| Restore into a new sandbox | New sandbox reaches running with the workspace | restored sandbox ID and restore response |
| Rerun the test after restore | Test still passes | restored command ID |
| Destroy restored sandbox | No tagged running task remains | final ECS listing and dashboard capture |

Run the same create, command, snapshot, restore, and destroy sequence through
the TypeScript SDK as a client-level check. The SDK must use the canonical
`/v1` contract; it must not call ECS, S3, or DynamoDB directly.

Use [`scripts/smoke-aws.sh`](../scripts/smoke-aws.sh) for the lower-level API
and infrastructure smoke path when debugging. It complements the agent run;
it does not replace it.

## Gate 3: AWS evidence

Use [the AWS operations runbook](aws/operations.md) and capture only redacted,
read-only evidence:

- ECS shows the control-plane service and both sandbox task transitions.
- CloudWatch contains the correlated request, sandbox, command, and runtime
  log streams without API keys, AWS credentials, or runtime tokens.
- S3 contains the snapshot object with server-side encryption and the expected
  checksum; Block Public Access remains enabled.
- DynamoDB contains the lifecycle and snapshot records when projected without
  the sensitive task field; tables are `ACTIVE` with the expected TTL/index
  settings.
- The final ECS listing contains no running task with either sandbox tag.
- The sandbox task has no public IP, host mount, Docker socket, or AWS
  credentials.

## Gate 4: failure paths

Exercise each case in a disposable development environment and retain the
request ID, typed error, and cleanup result:

| Failure | Expected behavior |
| --- | --- |
| Command timeout | Bounded timeout error; command and descendants terminate |
| Runtime health failure | Sandbox becomes unhealthy/failed and cleanup remains possible |
| S3 checksum mismatch | Restore is rejected; destination is not reported as usable |
| Duplicate create request | Idempotent response does not create a second sandbox |
| Duplicate destroy request | Idempotent terminal response; no new task or error loop |
| Restore into non-running sandbox | Typed state-conflict error; snapshot remains available |

Do not rerun a failed live case blindly. Correlate the request with CloudWatch,
DynamoDB, ECS, and S3, clean up the confirmed resources, and record the
remaining gap as a release decision.

## Submission packet and video

Store the redacted manifest and links to evidence outside the repository. The
three-minute video should follow this timing so AWS proves the product without
becoming the main actor:

| Time | Story beat |
| --- | --- |
| 0:00–0:20 | Agent receives the bug-fix request and calls MCP |
| 0:20–0:45 | Dashboard shows sandbox lifecycle and the agent clones the fixture |
| 0:45–1:15 | Failed test, live output, file edit, and passing retest |
| 1:15–1:35 | Snapshot plus brief S3/CloudWatch evidence |
| 1:35–1:55 | Original ECS task stops and a new sandbox restores the snapshot |
| 1:55–2:20 | Restored test passes and the second task is destroyed |
| 2:20–2:45 | Brief ECS/DynamoDB/no-credentials evidence |
| 2:45–3:00 | Return to the agent and state the disposable-computer value |

The video must not imply that a dry run, fake provider, or curl-only flow is a
verified coding-agent integration. See
[the hackathon scope decision](decisions/0004-hackathon-scope.md) for the
features intentionally outside this submission.

## Release decision

Ship only when every live gate is green, both sandboxes are destroyed, and the
evidence packet is redacted. If a live AWS or coding-agent prerequisite is not
available, keep this ticket open, publish the exact blocker and owner, and do
not mark the final rehearsal complete.
