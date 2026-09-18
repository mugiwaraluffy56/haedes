# haedes web

The public landing page explains the haedes computer abstraction and links
developers to the dashboard. It is intentionally a product-level surface: it
does not expose ECS, Fargate, S3, IAM, or task identifiers as user concepts.

## Local development

```sh
pnpm --filter web dev
pnpm --filter web typecheck
pnpm --filter web build
```

Set `NEXT_PUBLIC_DASHBOARD_URL` to point the calls to action at a deployed
dashboard. It defaults to `/dashboard` for local composition.
