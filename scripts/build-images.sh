#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
image_tag="${IMAGE_TAG:-dev}"
base_image="${BASE_IMAGE:-haedes-sandbox-base:${image_tag}}"
dev_image="${DEV_IMAGE:-haedes-sandbox-dev:${image_tag}}"

docker build \
  --platform linux/amd64 \
  --file "${repo_root}/images/sandbox-base/Dockerfile" \
  --tag "${base_image}" \
  "${repo_root}"

docker build \
  --platform linux/amd64 \
  --build-arg "BASE_IMAGE=${base_image}" \
  --file "${repo_root}/images/sandbox-dev/Dockerfile" \
  --tag "${dev_image}" \
  "${repo_root}"

printf 'Built %s and %s\n' "${base_image}" "${dev_image}"
