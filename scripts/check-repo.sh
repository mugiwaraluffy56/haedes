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
  CONTRIBUTING.md
  SECURITY.md
)

for file in "${required_files[@]}"; do
  test -f "$file" || { echo "missing required file: $file" >&2; exit 1; }
done

required_dirs=(
  apps/web apps/dashboard
  packages/sdk-typescript packages/api-types packages/shared-config packages/protocol
  services/control-plane services/lifecycle-manager services/snapshot-service services/sandbox-runtime
  integrations/github integrations/mcp integrations/agents integrations/examples
  infrastructure/terraform infrastructure/aws infrastructure/docker infrastructure/environments
  images/sandbox-base images/sandbox-dev
  internal/contracts internal/schemas internal/fixtures
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
