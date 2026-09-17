# Ownership and review matrix

Each boundary has one clear owner. Changes that cross boundaries should include tests and documentation for every affected owner.

| Area | Source boundary | Owner | Review requirements |
| --- | --- | --- | --- |
| Public UI | `apps/web`, `apps/dashboard` | TypeScript/UI owners | Verify user-facing behavior and dashboard API usage |
| TypeScript SDK | `packages/sdk-typescript` | TypeScript SDK owners | Review against the public `/v1` API contract |
| API types and runtime protocol | `packages/api-types`, `packages/protocol`, `internal/contracts`, `internal/schemas` | Contract owners | Run `make generate`; generated drift must fail |
| MCP adapter | `integrations/mcp` | MCP owners | Keep it thin; use the public API and never add AWS or lifecycle state |
| GitHub and agent adapters | `integrations/github`, `integrations/agents`, `integrations/examples` | Integration owners | Preserve provider-neutral boundaries |
| Control plane | `services/control-plane` | Go control-plane owners | Keep AWS calls behind interfaces and adapters |
| Lifecycle and snapshots | `services/lifecycle-manager`, `services/snapshot-service` | Go service owners | Preserve lifecycle ownership and cleanup semantics |
| Sandbox runtime | `services/sandbox-runtime` | Rust runtime owners | Preserve `/workspace` isolation, command limits, and runtime authentication |
| AWS infrastructure | `infrastructure/terraform`, `infrastructure/aws`, `images/` | Infrastructure owners | Terraform changes require a matching AWS/runbook update; never add credentials to tasks |
| Tests and operations | `tests/`, `scripts/`, `Makefile` | Repository maintainers | Keep checks runnable from a clean checkout |

MCP changes must be reviewed against the canonical public API contract. MCP must not call ECS, S3, DynamoDB, CloudWatch, or the Rust runtime directly, and it must not maintain an independent lifecycle or persistence boundary.
