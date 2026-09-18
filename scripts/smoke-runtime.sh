#!/usr/bin/env bash
set -euo pipefail

image="${IMAGE:-haedes-sandbox-dev:dev}"
container="haedes-runtime-smoke-$$"
cleanup() {
  docker rm --force "${container}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker run --rm --entrypoint /bin/bash "${image}" -euo pipefail -c '
  for tool in bash curl wget git python3 pip3 node npm go rustc cargo gcc g++ make; do
    command -v "${tool}" >/dev/null
  done
  test -w /workspace
  test -w /tmp
'

if docker run --rm --entrypoint /usr/local/bin/haedes-entrypoint "${image}" >/dev/null 2>&1; then
  echo "runtime unexpectedly started without HAEDES_RUNTIME_TOKEN" >&2
  exit 1
fi

docker run --detach --name "${container}" \
  --env HAEDES_RUNTIME_TOKEN=smoke-token \
  --env HAEDES_SANDBOX_ID=sbx_smoke \
  "${image}" >/dev/null

for attempt in {1..30}; do
  if docker exec "${container}" curl --fail --silent http://127.0.0.1:8080/healthz >/dev/null; then
    echo "runtime smoke test passed"
    exit 0
  fi
  sleep 1
done

echo "runtime did not become healthy" >&2
exit 1
