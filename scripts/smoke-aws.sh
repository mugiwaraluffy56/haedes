#!/usr/bin/env bash
set -euo pipefail

if [[ "${AWS_INTEGRATION_TESTS:-false}" != "true" ]]; then
  echo "AWS smoke test skipped; set AWS_INTEGRATION_TESTS=true to opt in"
  exit 0
fi

if [[ "${HAEDES_ENV:-}" != "development" && "${HAEDES_ENV:-}" != "test" ]]; then
  echo "AWS smoke tests require HAEDES_ENV=development or HAEDES_ENV=test" >&2
  exit 1
fi

api_base_url="${HAEDES_API_BASE_URL:?Set HAEDES_API_BASE_URL to the deployed control-plane URL}"
api_key="${HAEDES_API_KEY:?Set HAEDES_API_KEY to a deployed API key}"
image="${HAEDES_SANDBOX_IMAGE:-haedes-sandbox-dev:dev}"
headers=(-H "Authorization: Bearer ${api_key}" -H 'Content-Type: application/json')
idempotency="aws-smoke-$(date +%s)"

cleanup() {
  if [[ -n "${sandbox_id:-}" ]]; then
    curl --fail --silent --show-error -X DELETE "${headers[@]}" "${api_base_url}/v1/sandboxes/${sandbox_id}" >/dev/null || true
  fi
}
trap cleanup EXIT

curl --fail --silent --show-error "${api_base_url}/healthz" >/dev/null

create_response="$(curl --fail --silent --show-error "${headers[@]}" \
  -H "Idempotency-Key: ${idempotency}" \
  --data "{\"config\":{\"image\":\"${image}\",\"cpuMillis\":512,\"memoryMiB\":1024,\"storageGiB\":10,\"maxLifetimeSeconds\":600,\"defaultCommandTimeoutSeconds\":30,\"environment\":{\"SMOKE\":\"aws\"}}}" \
  "${api_base_url}/v1/sandboxes")"
sandbox_id="$(jq -er '.id' <<<"${create_response}")"
[[ "$(jq -r '.state' <<<"${create_response}")" == "running" ]]

command_response="$(curl --fail --silent --show-error "${headers[@]}" \
  --data '{"command":"printf smoke"}' \
  "${api_base_url}/v1/sandboxes/${sandbox_id}/commands")"
[[ "$(jq -r '.exitCode' <<<"${command_response}")" == "0" ]]

curl --fail --silent --show-error "${headers[@]}" -X PUT \
  --data-binary 'aws smoke file' \
  "${api_base_url}/v1/sandboxes/${sandbox_id}/files/content?path=/workspace/smoke.txt" >/dev/null
[[ "$(curl --fail --silent --show-error "${headers[@]}" \
  "${api_base_url}/v1/sandboxes/${sandbox_id}/files/content?path=/workspace/smoke.txt")" == "aws smoke file" ]]

snapshot_response="$(curl --fail --silent --show-error "${headers[@]}" \
  -X POST "${api_base_url}/v1/sandboxes/${sandbox_id}/snapshots")"
[[ "$(jq -r '.state' <<<"${snapshot_response}")" == "available" ]]

curl --fail --silent --show-error -X DELETE "${headers[@]}" \
  "${api_base_url}/v1/sandboxes/${sandbox_id}" >/dev/null
sandbox_id=""
echo "AWS adapter and sandbox smoke test passed"
