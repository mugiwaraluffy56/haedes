#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${repo_root}/infrastructure/docker/compose.yaml"
control_plane_port="${HAEDES_CONTROL_PLANE_PORT:-18080}"
api_key="${HAEDES_LOCAL_API_KEY:-local-api-key}"

cleanup() {
  if [[ -n "${restored_sandbox_id:-}" ]]; then
    curl --fail --silent --show-error -X DELETE -H "Authorization: Bearer ${api_key}" \
      "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${restored_sandbox_id}" >/dev/null || true
  fi
  if [[ -n "${sandbox_id:-}" ]]; then
    curl --fail --silent --show-error -X DELETE -H "Authorization: Bearer ${api_key}" \
      "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}" >/dev/null || true
  fi
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
case "${response}" in *'"state":"running"'*) ;; *) echo "control-plane smoke check failed: ${response}" >&2; exit 1 ;; esac
sandbox_id="$(jq -er '.id' <<<"${response}")"

command_response="$(curl --fail --silent --show-error \
  -H "Authorization: Bearer ${api_key}" -H 'Content-Type: application/json' \
  --data '{"command":"printf compose-journey"}' \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}/commands")"
command_id="$(jq -er '.id' <<<"${command_response}")"
[[ "$(jq -r '.exitCode' <<<"${command_response}")" == "0" ]]

events_file="$(mktemp)"
curl --fail --silent --show-error --max-time 15 \
  -H "Authorization: Bearer ${api_key}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}/commands/${command_id}/events" >"${events_file}"
grep -q 'event: completed' "${events_file}"
rm -f "${events_file}"

curl --fail --silent --show-error -X PUT \
  -H "Authorization: Bearer ${api_key}" -H 'Content-Type: application/octet-stream' \
  --data-binary 'compose snapshot file' \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}/files/content?path=/workspace/compose.txt" >/dev/null
[[ "$(curl --fail --silent --show-error -H "Authorization: Bearer ${api_key}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}/files/content?path=/workspace/compose.txt")" == "compose snapshot file" ]]
curl --fail --silent --show-error -H "Authorization: Bearer ${api_key}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}/files?path=/workspace" | jq -e '.entries | map(.path) | index("/workspace/compose.txt")' >/dev/null

snapshot_response="$(curl --fail --silent --show-error -X POST \
  -H "Authorization: Bearer ${api_key}" -H 'Idempotency-Key: local-compose-snapshot' \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}/snapshots")"
snapshot_id="$(jq -er '.id' <<<"${snapshot_response}")"
[[ "$(jq -r '.state' <<<"${snapshot_response}")" == "available" ]]

curl --fail --silent --show-error -X DELETE -H "Authorization: Bearer ${api_key}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${sandbox_id}" >/dev/null

restored_response="$(curl --fail --silent --show-error \
  -H "Authorization: Bearer ${api_key}" -H 'Content-Type: application/json' -H 'Idempotency-Key: local-compose-restore' \
  --data "{\"config\":{\"image\":\"haedes-sandbox-dev:dev\",\"cpuMillis\":512,\"memoryMiB\":1024,\"storageGiB\":10,\"maxLifetimeSeconds\":600,\"defaultCommandTimeoutSeconds\":30,\"environment\":{},\"snapshotId\":\"${snapshot_id}\"}}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes")"
restored_sandbox_id="$(jq -er '.id' <<<"${restored_response}")"
[[ "${restored_sandbox_id}" != "${sandbox_id}" ]]
[[ "$(curl --fail --silent --show-error -H "Authorization: Bearer ${api_key}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${restored_sandbox_id}/files/content?path=/workspace/compose.txt")" == "compose snapshot file" ]]

curl --fail --silent --show-error -X DELETE -H "Authorization: Bearer ${api_key}" \
  "http://127.0.0.1:${control_plane_port}/v1/sandboxes/${restored_sandbox_id}" >/dev/null
restored_sandbox_id=""
sandbox_id=""

echo "local compose smoke test passed"
