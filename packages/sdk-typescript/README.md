# @haedes/sdk

The TypeScript SDK provides a product-focused interface to the haedes sandbox
API. It hides ECS, Fargate, S3, DynamoDB, and other provider details.

```ts
import { SandboxClient } from '@haedes/sdk';

const client = new SandboxClient({
  baseUrl: process.env.HAEDES_API_URL!,
  apiKey: process.env.HAEDES_API_KEY!,
});

const sandbox = await client.sandboxes.create({
  image: 'haedes-sandbox-dev:dev',
  environment: { PROJECT: 'demo' },
});

const result = await sandbox.exec('printf hello');
console.log(result.exitCode);

for await (const event of sandbox.execStream('npm test')) {
  if (event.type === 'stdout' || event.type === 'stderr') {
    process.stdout.write(event.data);
  }
  if (event.type === 'failed') {
    console.error(`${event.code}: ${event.message}`);
  }
}

await sandbox.writeFile('/workspace/README.md', '# demo');
console.log(await sandbox.readFile('/workspace/README.md'));

const snapshot = await sandbox.snapshot({ expiresInSeconds: 3600 });
await sandbox.destroy();

const restored = await client.sandboxes.create({ snapshotId: snapshot.id });
console.log(await restored.readFile('/workspace/README.md'));
await restored.destroy();
```

Pass a custom `fetch` implementation and `requestTimeoutMs` when integrating
with a runtime that needs its own transport or timeout policy.

`execStream` yields ordered command events from the public SSE endpoint and
automatically resumes once after a dropped connection using the last received
event ID.
