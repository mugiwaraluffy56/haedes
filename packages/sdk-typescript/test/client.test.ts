import assert from 'node:assert/strict';
import { SandboxClient } from '../src/index.js';

const calls: Array<{ url: string; init: RequestInit }> = [];
const client = new SandboxClient({
  baseUrl: 'https://api.example.test/',
  apiKey: 'secret-key',
  fetch: async (input, init = {}) => {
    calls.push({ url: String(input), init });
    if (String(input).includes('/v1/sandboxes') && !String(input).includes('/v1/sandboxes/sbx_1')) {
      if (init.method === 'POST') {
        return Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] }, { status: 202 });
      }
      return Response.json({ items: [], page: { hasMore: false } });
    }
    if (String(input).endsWith('/v1/sandboxes/sbx_1')) {
      return init.method === 'DELETE'
        ? new Response(null, { status: 202 })
        : Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] });
    }
    throw new Error(`unexpected request: ${String(input)}`);
  },
});

const created = await client.sandboxes.create({ image: 'sandbox:test' });
assert.equal(created.id, 'sbx_1');
assert.equal(calls[0].url, 'https://api.example.test/v1/sandboxes');
assert.equal(new Headers(calls[0].init.headers).get('Authorization'), 'Bearer secret-key');
assert.ok(new Headers(calls[0].init.headers).get('Idempotency-Key'));
assert.deepEqual(JSON.parse(String(calls[0].init.body)), {
  config: {
    image: 'sandbox:test',
    cpuMillis: 512,
    memoryMiB: 1024,
    storageGiB: 10,
    maxLifetimeSeconds: 600,
    defaultCommandTimeoutSeconds: 60,
    environment: {},
  },
});

const page = await client.sandboxes.list({ limit: 10, state: 'running' });
assert.deepEqual(page, { items: [] });
const fetched = await client.sandboxes.get('sbx_1');
assert.equal((await fetched.get()).id, 'sbx_1');
await fetched.destroy();
assert.equal(calls.at(-1)?.init.method, 'DELETE');

console.log('sdk client checks passed');
