# haedes project context

## Current vision

**Give your AI agent a computer on AWS.**

haedes provides disposable AWS computers that agents can create, use, snapshot, restore, and destroy. Compute is temporary; workspace state can survive through snapshots and continue on a new sandbox.

## Integration layers

1. Agent-native integration: MCP is the first adapter.
2. Developer integration: the TypeScript SDK.
3. Universal platform contract: the versioned HTTP `/v1` API.

All three surfaces converge on the Go control plane. AWS orchestration provisions private Fargate tasks running the Rust runtime, which owns commands and `/workspace`.

## Hackathon proof

The primary demo begins inside one real coding agent. It creates a sandbox through MCP, fixes a failing repository in the remote computer, streams tests, snapshots the workspace, destroys the task, restores the snapshot into a new task, and proves the tests still pass.
