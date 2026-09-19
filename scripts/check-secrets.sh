#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

command -v gitleaks >/dev/null 2>&1 || {
  echo "gitleaks is required for secret scanning" >&2
  exit 1
}

gitleaks detect --source . --no-banner --redact --exit-code 1

echo "secret scan passed"
