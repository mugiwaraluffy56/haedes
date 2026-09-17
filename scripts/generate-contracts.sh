#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

mode="${1:---check}"
case "$mode" in
  --check|--write) ;;
  *) echo "usage: $0 [--check|--write]" >&2; exit 2 ;;
esac

api_contract="internal/contracts/api.openapi.yaml"
generated_types="packages/api-types/src/generated.ts"

# The public contract is introduced by issue #3. Keep this command runnable
# before then, while making the eventual drift check part of every root check.
if [[ ! -f "$api_contract" ]]; then
  echo "no canonical API contract yet; generation check skipped"
  exit 0
fi

if [[ ! -f "$generated_types" ]]; then
  echo "canonical API contract found; generated output will be checked after issue #5 adds $generated_types"
  exit 0
fi

if ! git ls-files --error-unmatch "$generated_types" >/dev/null 2>&1; then
  echo "generated contract output must be tracked: $generated_types" >&2
  exit 1
fi

temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT

pnpm exec openapi-typescript "$api_contract" -o "$temporary_dir/generated.ts"

if [[ "$mode" == "--write" ]]; then
  cp "$temporary_dir/generated.ts" "$generated_types"
  echo "generated $generated_types"
  exit 0
fi

if ! cmp -s "$temporary_dir/generated.ts" "$generated_types"; then
  echo "generated contract drift detected in $generated_types" >&2
  diff -u "$generated_types" "$temporary_dir/generated.ts" || true
  echo "run 'make generate-write' to update generated output" >&2
  exit 1
fi

echo "generated contracts are up to date"
