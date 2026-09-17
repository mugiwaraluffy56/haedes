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

## Git workflow

- Every task must be implemented on a new branch created from the latest `master`.
- Use a descriptive branch name such as `feat/runtime-command-execution` or `fix/auth-error-mapping`.
- Do not commit issue implementation changes directly to `master` or push them directly to `master`.
- Every completed task must be submitted as a pull request for review before it is merged.
- Pull requests must describe the change, link the relevant GitHub issue, and report the checks that were run.
- Explicitly requested repository workflow or instruction updates may be committed and pushed directly to `master`.

### After a pull request is merged

Keep the checkout on the main branch and remove completed task branches:

```sh
git fetch origin --prune
git switch master
git pull --ff-only origin master
git branch -d <merged-branch>
git push origin --delete <merged-branch>
git fetch origin --prune
```

Before starting another issue, always update `master` first and create a fresh branch from it:

```sh
git fetch origin --prune
git switch master
git pull --ff-only origin master
git switch -c <descriptive-branch-name>
```

Never continue issue work on a merged branch or reuse an old task branch.

The implementation board is in [`docs/TICKETS.md`](docs/TICKETS.md); each ticket links to its GitHub issue.
