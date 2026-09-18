#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

command -v docker >/dev/null 2>&1 || {
  echo "docker is required for container checks" >&2
  exit 1
}
if ! docker info >/dev/null 2>&1; then
  echo "docker daemon is unavailable; start Docker before running container checks" >&2
  exit 1
fi

docker compose -f infrastructure/docker/compose.yaml config --quiet

image_tag="${IMAGE_TAG:-ci}"
IMAGE_TAG="$image_tag" ./scripts/build-images.sh
docker build \
  --file infrastructure/docker/control-plane.Dockerfile \
  --tag "haedes-control-plane:${image_tag}" \
  .

IMAGE="haedes-sandbox-dev:${image_tag}" ./scripts/smoke-runtime.sh

echo "container quality checks passed"
