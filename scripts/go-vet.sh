#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
services=(control-plane lifecycle-manager snapshot-service)

for service in "${services[@]}"; do
  service_dir="$repo_root/services/$service"
  if ! packages="$(cd "$service_dir" && go list ./... 2>/dev/null)"; then
    echo "$service: unable to enumerate Go packages" >&2
    exit 1
  fi
  if [[ -n "$packages" ]]; then
    (cd "$service_dir" && go vet ./...)
  else
    echo "$service: no Go packages yet; manifest is valid"
  fi
done
