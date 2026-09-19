#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

./scripts/check-repo.sh
pnpm lint
pnpm typecheck
pnpm test
./scripts/go-vet.sh
./scripts/go-test.sh
cargo fmt --all -- --check
cargo clippy --workspace --all-targets --all-features -- -D warnings
cargo test --workspace

echo "core quality checks passed"
