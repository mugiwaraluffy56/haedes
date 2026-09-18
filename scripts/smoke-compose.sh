#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${repo_root}/infrastructure/docker/compose.yaml"
control_plane_port="${HAEDES_CONTROL_PLANE_PORT:-18080}"
api_key="${HAEDES_LOCAL_API_KEY:-local-api-key}"

cleanup() {
  docker compose -f "${compose_file}" down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -f "${compose_file}" config --quiet
docker compose -f "${compose_file}" up --detach --wait

curl --fail --silent --show-error --retry 20 --retry-delay 1 \
  "http://127.0.0.1:${control_plane_port}/healthz" >/dev/null

runtime_health="$(docker compose -f "${compose_file}" exec --no-TTY control-plane \
  curl --fail --silent http://sandbox-runtime:8080/healthz)"
case "${runtime_health}" in
  *'"status":"ok"'*) ;;
  *) echo "runtime health check failed: ${runtime_health}" >&2; exit 1 ;;
esac

response="$(curl --fail --silent --show-error \
  -H "Authorization: Bearer ${api_key}" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: local-smoke-create' \
  --data '{"config":{"image":"haedes-sandbox-dev:dev","cpuMillis":512,"memoryMiB":1024,"storageGiB":10,"maxLifetimeSeconds":600,"defaultCommandTimeoutSeconds":30,"environment":{"MODE":"local"}}}' \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes")"
case "${response}" in
  *'"state":"running"'*) ;;
  *) echo "control-plane smoke check failed: ${response}" >&2; exit 1 ;;
esac

echo "local compose smoke test passed"
