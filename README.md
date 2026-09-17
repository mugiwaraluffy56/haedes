# haedes

## Give your AI agent a computer on AWS.

haedes is an execution platform for AI agents. It gives an agent a temporary, isolated Linux computer on AWS so the agent can clone a repository, run commands, inspect and modify files, save its workspace, and continue later on a new computer.

The agent is the brain. haedes is the computer.

## Architecture

```text
Agent client → MCP adapter → TypeScript SDK (when useful) → HTTP /v1 API
             → Go control plane → ECS/Fargate → Rust runtime → /workspace
```

The HTTP API is the canonical platform contract. MCP and the SDK are thin integration surfaces; the dashboard provides human observability and control.

## Source of truth

- [Product requirements](docs/prd.md)
- [Implementation plan](docs/plan.md)
- [Implementation tickets](docs/tickets.md)

## Repository foundation

The repository is a bounded multi-language monorepo:

- TypeScript applications, packages, MCP, and examples use pnpm workspaces and Turborepo.
- Go services are explicit members of the root `go.work`.
- The Rust sandbox runtime is the root Cargo workspace member.
- `services/control-plane` owns orchestration; `services/sandbox-runtime` owns process and `/workspace` behavior.
- `integrations/mcp` is a thin adapter over the public API and does not call AWS directly.

Run `make check` for the repository foundation checks. See [the repository architecture](docs/architecture/repository.md) and [contributing guide](CONTRIBUTING.md) for ownership rules.

## Hackathon demo

One real coding agent requests a sandbox through MCP, fixes a failing authentication test inside the Fargate computer, snapshots the workspace, destroys the task, restores the snapshot into a new task, and reruns the tests successfully.

## Status

The repository is being built from the implementation plan. See the linked tickets for the current work breakdown and dependencies.
