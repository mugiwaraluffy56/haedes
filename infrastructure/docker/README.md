# Docker boundary

Local service composition and development images belong here. Docker is for development and testing, not a user-facing sandbox mode.

## Local workflow

Build the sandbox image and start the local control plane plus runtime:

```sh
./scripts/build-images.sh
docker compose -f infrastructure/docker/compose.yaml up --build --wait
```

Run the repeatable health and authenticated lifecycle smoke test with:

```sh
./scripts/smoke-compose.sh
```

The control plane uses the deterministic in-memory adapters from
`services/control-plane/internal/fakes`. The runtime container is still the
same non-root image and token contract used by AWS tasks; the local fake
control-plane adapter points at the runtime's internal Compose DNS name so the
runtime is never published to the host.

The Compose network is internal, the runtime has no host port, and both
containers drop capabilities, use `no-new-privileges`, and receive no AWS
credentials, host mounts, or Docker socket. The local API key is intentionally
development-only and can be overridden with `HAEDES_LOCAL_API_KEY`.
