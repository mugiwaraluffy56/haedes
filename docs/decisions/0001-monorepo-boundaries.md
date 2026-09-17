# ADR 0001: Bounded multi-language monorepo

- Status: accepted
- Date: 2026-09-17

## Context

haedes has TypeScript clients and applications, Go control-plane services, and a Rust sandbox runtime. These components share one product but have different build tools and ownership boundaries.

## Decision

Use pnpm and Turborepo for JavaScript workspaces, a root Go workspace for named Go services, and a root Cargo workspace for Rust services. Keep MCP under `integrations/mcp` as a TypeScript adapter over the public HTTP API. Keep GitHub and provider-neutral adapter boundaries explicit without registering them as packages until they own package code.

## Consequences

Each service can evolve and test independently while repository-wide checks remain discoverable from the root. AWS orchestration stays in the control plane, runtime behavior stays in Rust, and no catch-all backend can absorb unrelated responsibilities.
