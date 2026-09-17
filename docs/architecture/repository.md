# Repository architecture

The repository is a deliberately bounded monorepo. Each top-level area owns one kind of change:

| Boundary | Responsibility |
| --- | --- |
| `apps/` | Human-facing web and dashboard applications |
| `packages/` | Shared TypeScript libraries and schemas |
| `services/control-plane` | Go public API, lifecycle, authorization, and AWS orchestration |
| `services/lifecycle-manager` | Go expiration and reconciliation worker |
| `services/snapshot-service` | Go snapshot/archive coordination boundary |
| `services/sandbox-runtime` | Rust process and `/workspace` filesystem runtime |
| `integrations/mcp` | Thin MCP adapter over the public API/SDK |
| `integrations/github` | Repository URL and credential boundary |
| `integrations/agents` | Provider-neutral future adapter contracts |
| `infrastructure/` and `images/` | Deployment, local composition, and sandbox image definitions |
| `internal/` | Canonical contracts, schemas, and fixtures |
| `tests/` | Integration, end-to-end, security, and load suites |
| `docs/` | Architecture, API, sandbox, AWS, security, and decision records |

JavaScript workspaces are discovered only from `apps/*`, `packages/*`, `integrations/mcp`, and `integrations/examples/*`. Go services are explicit members of the root `go.work`; the Rust runtime is the only current Cargo member. This keeps provider-specific code behind adapters and prevents a generic backend or unowned integration directory from becoming an accidental extension point.
