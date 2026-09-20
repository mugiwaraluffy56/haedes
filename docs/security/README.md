# Security and threat model

Treat agent prompts, repository contents, commands, environment values, and
snapshot data as untrusted. The security boundary is the control plane and its
adapters, not the coding agent.

## Trust boundaries

| Boundary | Threat | Control |
| --- | --- | --- |
| Agent/MCP to public API | Forged identity, undocumented fields, credential injection | Server-side API key, owner authorization, strict tool schemas, bounded inputs, typed errors |
| Control plane to runtime | Public credential or cross-sandbox access reaching a task | Short-lived sandbox-scoped runtime token; public API keys are never forwarded |
| Runtime to host | Path traversal, symlink escape, runaway process, privilege escalation | `/workspace` path guard, non-root UID 10001, read-only root filesystem, command/output/time limits, process-group cleanup |
| Control plane to AWS | Excessive cloud permissions or task impersonation | AWS calls behind adapters; task role is scoped to ECS lifecycle, metadata, snapshot objects, and metrics; `iam:PassRole` names only the sandbox execution role |
| Snapshot storage | Public disclosure or archive tampering | S3 Block Public Access, BucketOwnerEnforced ownership, AES-256 SSE, versioning, lifecycle expiration, SHA-256 verification, private IAM access |
| Network | Direct public access to sandbox tasks | ALB is the public entry point; control plane and sandbox tasks run in private subnets; sandbox ingress allows port 8080 only from the control-plane security group |
| Images and dependencies | Untrusted or mutable runtime image | ECR immutable tags, scan-on-push, digest-pinned production image inputs, CI dependency and secret checks |

## Credential rules

- AWS credentials exist only in the control-plane task role. Sandbox tasks have
  no AWS task role and receive no AWS environment variables.
- `HAEDES_API_KEY` stays in the MCP server process and is never a tool
  argument, prompt value, sandbox environment entry, or tool result.
- Runtime tokens and private task endpoints are implementation secrets. Do not
  print the `task` field from DynamoDB records or paste it into tickets.
- API signing material is configured through the deployment secret mechanism;
  never commit real values to `.env`, Terraform variables, logs, or fixtures.

## Workspace and lifecycle rules

The Rust runtime owns `/workspace` and rejects paths that resolve outside it,
including symlink escapes. Commands run as the unprivileged `sandbox` user and
are bounded to 16 KiB input, 1 MiB output, 32 environment entries, and a
15-minute timeout. File bodies are limited to 10 MiB and compressed snapshot
archives to 100 MiB.

The Go control plane is the only lifecycle owner. Expiration, explicit destroy,
failed provisioning, and restore all converge on the same state machine; a
runtime health response or ECS status is an input, not an alternate public
state. See [the lifecycle contract](../../internal/contracts/lifecycle.md).

## Operational response

If a task, token, object, or log appears exposed:

1. Stop the affected sandbox through the authenticated API and record its
   sandbox ID and request ID.
2. Stop the ECS task if API cleanup cannot reach it; preserve CloudWatch logs
   and the relevant DynamoDB item for investigation.
3. Rotate the API/signing secret or AWS role credentials according to the
   affected boundary. Never delete evidence before the incident owner approves.
4. Check S3 object versions, DynamoDB ownership, ECS task tags, and CloudWatch
   streams for neighboring sandboxes.
5. File a private security report with the reproduction, boundary, timestamps,
   affected resource IDs, and cleanup actions. Do not use a public issue.

The repository checks reinforce these rules with boundary tests, gitleaks,
dependency audit, Terraform assertions, and container scanning.
