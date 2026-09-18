# Integration tests

Cross-service tests using local fakes or containers belong here.

The SDK/MCP/dashboard journey runs without external services:

```sh
pnpm test:integration
```

The local control-plane/runtime container journey is opt-in and covers command
events, workspace files, snapshot round-trip, restore, and cleanup:

```sh
./scripts/build-images.sh
./scripts/smoke-compose.sh
```

The AWS smoke path is separately gated by `AWS_INTEGRATION_TESTS=true` and an
explicit `HAEDES_ENV=development` or `HAEDES_ENV=test`; it must never be used
as a production health check.
