# Contributing to haedes

## Boundaries

haedes is an execution platform that gives AI agents temporary Linux computers on AWS. The agent is the brain; the sandbox is the computer.

- Public UI and SDK work belongs in `apps/` and `packages/`.
- MCP transport, tool schemas, validation, and adapter behavior belong in `integrations/mcp`.
- Sandbox identity, lifecycle, authorization, and AWS orchestration belong in `services/control-plane`.
- Process execution and `/workspace` filesystem behavior belong in `services/sandbox-runtime`.
- Expiration and snapshot workers belong in their named Go services.
- AWS changes require both Terraform changes and an updated runbook under `infrastructure/aws` or `docs/aws`.

MCP must use the public HTTP API. It must not call ECS, S3, DynamoDB, CloudWatch, or the Rust runtime directly, and it must not maintain independent lifecycle state.

## Local checks

Use Node 20.18+, pnpm 9.15+, Go 1.23+, and Rust 1.80+. From the repository root:

```sh
make check
```

Focused checks are also available as `pnpm lint`, `pnpm typecheck`, `pnpm test`, `./scripts/go-test.sh`, and `cargo test --workspace`. Go service modules can be tested directly with `cd services/<name> && go test ./...`.

## Changes

Create a branch from the latest `main`. Keep commits focused, update the relevant source-of-truth documentation, and open a pull request before merging. Pull requests should describe the change, link the relevant GitHub issue, and report the checks that were run.
