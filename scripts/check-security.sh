#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

pnpm test:security
pnpm test:load
./scripts/check-dependencies.sh
./scripts/check-secrets.sh

echo "security quality checks passed"
