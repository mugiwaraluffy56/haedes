# Contributing to haedes

haedes is an execution platform that gives AI agents temporary Linux computers on AWS.

## Product boundary

- The agent is the brain; the sandbox is the computer.
- One initial sandbox maps to one temporary private ECS/Fargate task.
- The Go control plane owns lifecycle and AWS orchestration.
- The Rust runtime owns processes and `/workspace`.
- The HTTP `/v1` API is the canonical platform contract.
- The TypeScript SDK and MCP server are client adapters over that contract.
- The dashboard is a human observability and control surface.

## Working rules

- Keep provider-specific implementation behind interfaces and adapters.
- Never put AWS credentials in sandbox tasks.
- Preserve `/workspace` isolation, command limits, timeouts, and automatic cleanup.
- MCP must not call ECS directly or maintain independent lifecycle state.
- Keep the hackathon scope narrow: one MCP path, one real coding-agent demo, one AWS region, and one default image.
- Run the repository checks relevant to every changed boundary before opening a pull request.

The implementation board is in [`docs/tickets.md`](docs/tickets.md); each ticket links to its GitHub issue.
