# Contributing to haedes

## Boundaries

haedes is an execution platform that gives AI agents temporary Linux computers on AWS. The agent is the brain; the sandbox is the computer.

- Public UI work belongs in `apps/web` and `apps/dashboard`; TypeScript SDK work belongs in `packages/sdk-typescript`.
- Public API types and runtime protocol changes belong in `packages/api-types` and `packages/protocol` and must be reviewed against their canonical contract sources.
- MCP transport, tool schemas, validation, and adapter behavior belong in `integrations/mcp`.
- Sandbox identity, lifecycle, authorization, and AWS orchestration belong in `services/control-plane`.
- Process execution and `/workspace` filesystem behavior belong in `services/sandbox-runtime`.
- Expiration and snapshot workers belong in their named Go services.
- AWS changes require Terraform changes plus an updated runbook under `infrastructure/aws` or `docs/aws`.

MCP must use the public HTTP API. It must not call ECS, S3, DynamoDB, CloudWatch, or the Rust runtime directly, and it must not maintain independent lifecycle state.

See the [ownership matrix](docs/architecture/ownership.md) for the review boundary of each area.

## Local checks

Use Node 20.18+, pnpm 9.15+, Go 1.23+, and Rust 1.80+. From the repository root:

```sh
make check
```

`make check` is the complete local pull-request gate. It includes the language,
contract, container, Terraform, dependency, secret, security, and bounded-load
checks. The gates can also be run independently when iterating:

```sh
make check-core       # TypeScript, Go, and Rust formatting/checks/tests
make check-contracts  # Generated API output and schema validation
make check-containers # Docker builds, compose validation, and runtime smoke
make check-terraform  # Formatting, offline init, and validation for all stacks
make check-security   # Boundary/load tests, dependency audit, and secret scan
```

The complete gate requires Docker, Terraform, `cargo-audit`, and `gitleaks` in
addition to the language toolchains. It never applies Terraform or calls AWS;
the AWS smoke test remains opt-in through `AWS_INTEGRATION_TESTS=true`.

The root commands are intentionally repeatable from a clean checkout:

| Command | Purpose |
| --- | --- |
| `make check` | Complete local pull-request quality gate |
| `make test` | TypeScript, Go, and Rust tests |
| `make lint` | TypeScript lint, Go vet, and Rust formatting validation |
| `make format` | Apply Rust formatting |
| `make format-check` | Verify Rust formatting without changing files |
| `make typecheck` | Run all workspace type checks |
| `make generate` | Verify generated contract output matches its canonical source |
| `make generate-write` | Regenerate checked-in contract output after an intentional source change |

Focused checks are also available as `pnpm lint`, `pnpm typecheck`, `pnpm test`, `./scripts/go-test.sh`, and `cargo test --workspace`. Go service modules can be tested directly with `cd services/<name> && go test ./...`.

## Changes

Create a branch from the latest `main`. Keep commits focused, update the relevant source-of-truth documentation, and open a pull request before merging. Pull requests should describe the change, link the relevant GitHub issue, and report the checks that were run.
