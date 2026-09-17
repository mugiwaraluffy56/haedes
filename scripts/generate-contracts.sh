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
generated_go="services/control-plane/internal/contracts/generated.go"

# The public contract is introduced by issue #3. Keep this command runnable
# before then, while making the eventual drift check part of every root check.
if [[ ! -f "$api_contract" ]]; then
  echo "no canonical API contract yet; generation check skipped"
  exit 0
fi

for generated_file in "$generated_types" "$generated_go"; do
  if [[ "$mode" == "--check" ]]; then
    if [[ ! -f "$generated_file" ]]; then
      echo "generated contract output is missing: $generated_file" >&2
      exit 1
    fi
  fi
done

temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT

pnpm exec openapi-typescript "$api_contract" -o "$temporary_dir/generated.ts"
node scripts/generate-go-models.mjs "$api_contract" "$temporary_dir/generated.go"

if [[ "$mode" == "--write" ]]; then
  cp "$temporary_dir/generated.ts" "$generated_types"
  cp "$temporary_dir/generated.go" "$generated_go"
  echo "generated $generated_types and $generated_go"
  exit 0
fi

if ! cmp -s "$temporary_dir/generated.ts" "$generated_types"; then
  echo "generated contract drift detected in $generated_types" >&2
  diff -u "$generated_types" "$temporary_dir/generated.ts" || true
  echo "run 'make generate-write' to update generated output" >&2
  exit 1
fi

if ! cmp -s "$temporary_dir/generated.go" "$generated_go"; then
  echo "generated contract drift detected in $generated_go" >&2
  diff -u "$generated_go" "$temporary_dir/generated.go" || true
  echo "run 'make generate-write' to update generated output" >&2
  exit 1
fi

echo "generated TypeScript and Go contracts are up to date"
