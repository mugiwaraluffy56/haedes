# Agent adapter contracts

This directory contains the provider-neutral host contract used by examples
and future MCP adapters. `WorkspaceSession` wraps the TypeScript SDK for
create, command streaming, bounded file access, snapshots, restore, and
destruction.

The session is intentionally not an agent framework: it does not choose tools,
call a model, persist independent lifecycle state, or implement a planner. A
real coding agent remains the decision-maker and receives this host contract as
an integration boundary.
