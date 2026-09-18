# Sandbox development image

The development image is built on `haedes-sandbox-base` and is the default image used by local runtime smoke tests and AWS Fargate task definitions. Build both images with `scripts/build-images.sh`, then run `scripts/smoke-runtime.sh` against `haedes-sandbox-dev:dev`.

The image intentionally contains no privileged settings, host mounts, Docker socket, or AWS credentials. Local compose wiring belongs to issue #21.
