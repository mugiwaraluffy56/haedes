# Canonical contracts

Versioned public API and lifecycle contract sources belong here. The HTTP `/v1` API is the platform contract for all adapters.

Run `make generate` to refresh the TypeScript and Go-facing models, and run `pnpm validate:contracts` to check limits, identifiers, timestamps, lifecycle transitions, persisted schemas, runtime fixtures, and MCP references.
