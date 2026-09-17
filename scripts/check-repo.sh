#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

required_files=(
  package.json
  pnpm-workspace.yaml
  turbo.json
  go.work
  Cargo.toml
  Makefile
  .env.example
  .nvmrc
  pnpm-lock.yaml
  Cargo.lock
  CONTRIBUTING.md
  SECURITY.md
  internal/contracts/api.openapi.yaml
  internal/contracts/lifecycle.md
  internal/schemas/sandbox.schema.json
  internal/schemas/command.schema.json
  internal/schemas/snapshot.schema.json
  packages/protocol/src/runtime-v1.schema.json
  packages/api-types/src/generated.ts
  packages/api-types/src/index.ts
  services/control-plane/internal/contracts/generated.go
  integrations/mcp/src/tool-contract.schema.json
  docs/architecture/repository.md
  docs/architecture/ownership.md
  docs/architecture/state-machine.md
  docs/api/public-api.md
  docs/api/runtime-protocol.md
  docs/decisions/0001-monorepo-boundaries.md
  scripts/generate-contracts.sh
  scripts/generate-go-models.mjs
  scripts/validate-contracts.ts
)

for file in "${required_files[@]}"; do
  test -f "$file" || { echo "missing required file: $file" >&2; exit 1; }
done

workspace_packages=(
  apps/web
  apps/dashboard
  packages/sdk-typescript
  packages/api-types
  packages/shared-config
  packages/protocol
  integrations/mcp
  integrations/examples/basic-agent
  integrations/examples/coding-agent
)

for package_dir in "${workspace_packages[@]}"; do
  manifest="$package_dir/package.json"
  test -f "$manifest" || { echo "missing workspace manifest: $manifest" >&2; exit 1; }
  node -e '
    const fs = require("node:fs");
    const manifest = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
    for (const script of ["lint", "test", "typecheck"]) {
      if (!manifest.scripts?.[script]) {
        throw new Error(`${process.argv[1]} is missing the ${script} script`);
      }
    }
  ' "$manifest"
done

for module in services/control-plane services/lifecycle-manager services/snapshot-service; do
  test -f "$module/go.mod" || { echo "missing Go module: $module/go.mod" >&2; exit 1; }
done

test -f services/sandbox-runtime/Cargo.toml || {
  echo "missing Rust workspace member manifest: services/sandbox-runtime/Cargo.toml" >&2
  exit 1
}

for ownership_term in "Public UI" "TypeScript SDK" "MCP" "control-plane" "sandbox-runtime" "Terraform"; do
  rg -q "$ownership_term" CONTRIBUTING.md || {
    echo "ownership documentation is missing: $ownership_term" >&2
    exit 1
  }
done

required_dirs=(
  apps/web apps/dashboard
  packages/sdk-typescript packages/api-types packages/shared-config packages/protocol packages/protocol/src/fixtures
  services/control-plane services/lifecycle-manager services/snapshot-service services/sandbox-runtime
  integrations/github integrations/mcp integrations/agents integrations/examples
  infrastructure/terraform infrastructure/aws infrastructure/docker infrastructure/environments
  images/sandbox-base images/sandbox-dev
  internal/contracts internal/schemas internal/schemas/fixtures internal/fixtures
  tests/integration tests/e2e tests/security tests/load
  docs/architecture docs/api docs/sandbox docs/aws docs/security docs/decisions
  examples/basic-agent examples/coding-agent examples/github-fixer
)

for dir in "${required_dirs[@]}"; do
  test -d "$dir" || { echo "missing required directory: $dir" >&2; exit 1; }
done

if find . -maxdepth 2 -type d \( -name backend -o -name backends \) -print -quit | grep -q .; then
  echo "generic backend directories are not allowed; use a named service boundary" >&2
  exit 1
fi

echo "repository foundation is valid"
