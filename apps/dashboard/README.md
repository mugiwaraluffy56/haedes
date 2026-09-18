# haedes dashboard

The dashboard is the human observability and control surface for sandbox
lifecycles. It reads real data from the public `/v1` API through the
TypeScript SDK; it does not substitute local mock data when the API is absent.

## Configuration

Set these public development variables before starting the app:

```sh
export NEXT_PUBLIC_HAEDES_API_URL=http://localhost:8080
export NEXT_PUBLIC_HAEDES_API_KEY=development-key
```

## Local development

```sh
pnpm --filter dashboard dev
pnpm --filter dashboard typecheck
pnpm --filter dashboard lint
pnpm --filter dashboard build
```

The list refreshes every five seconds while any sandbox is in a non-terminal
state. Stopped, failed, and destroyed sandboxes do not cause polling.

The sandbox detail view is the human control surface for live work: it streams
stdout and stderr from SDK command events, records the command timeline for the
current session, shows derived AWS task status, and exposes snapshot, restore,
and destroy actions. The SDK owns SSE reconnect behavior and the dashboard does
not maintain a separate lifecycle state.
