#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

for command_name in pnpm go cargo cargo-audit; do
  command -v "$command_name" >/dev/null 2>&1 || {
    echo "$command_name is required for dependency checks" >&2
    exit 1
  }
done

pnpm audit --audit-level high

for module in services/control-plane services/lifecycle-manager services/snapshot-service; do
  (cd "$module" && go mod verify)
done

cargo audit --locked

echo "dependency checks passed"
