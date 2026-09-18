#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

command -v terraform >/dev/null 2>&1 || {
  echo "terraform is required for infrastructure checks" >&2
  exit 1
}

terraform fmt -check -recursive infrastructure/terraform

temporary_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$temporary_dir"
}
trap cleanup EXIT

cp -R infrastructure/terraform "$temporary_dir/terraform"

for directory in \
  "$temporary_dir/terraform" \
  "$temporary_dir/terraform/environments/dev" \
  "$temporary_dir/terraform/environments/prod"; do
  terraform -chdir="$directory" init -backend=false -input=false -no-color
  terraform -chdir="$directory" validate -no-color
done

echo "Terraform quality checks passed"
