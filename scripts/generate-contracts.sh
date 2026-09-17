#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Contract generators are added with the canonical API contract in Task 2.
# Keeping this entry point deterministic lets CI call it from the foundation.
echo "no generated contracts yet; canonical contract work starts in Task 2"
