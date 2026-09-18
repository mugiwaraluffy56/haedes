import assert from 'node:assert/strict';
import { SandboxClient, SandboxError } from '../src/index.js';

assert.throws(() => new SandboxClient({ baseUrl: '', apiKey: 'key' }), /baseUrl is required/);
assert.throws(() => new SandboxClient({ baseUrl: 'http://localhost', apiKey: '' }), /apiKey is required/);

const client = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  fetch: async () => new Response(JSON.stringify({ error: { code: 'not_found', message: 'Missing sandbox.', requestId: 'req_1', details: { resource: 'sandbox' } } }), {
    status: 404,
    headers: { 'Content-Type': 'application/json', 'X-Request-ID': 'req_1' },
  }),
});

await assert.rejects(client.sandboxes.get('sbx_missing'), (error: unknown) => {
  assert.ok(error instanceof SandboxError);
  assert.equal(error.code, 'not_found');
  assert.equal(error.requestId, 'req_1');
  assert.equal(error.status, 404);
  return true;
});

const timeoutClient = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  requestTimeoutMs: 5,
  fetch: async (_input, init = {}) => new Promise<Response>((_resolve, reject) => {
    init.signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true });
  }),
});
await assert.rejects(timeoutClient.sandboxes.get('sbx_slow'), (error: unknown) => {
  assert.ok(error instanceof SandboxError);
  assert.equal(error.code, 'request_timeout');
  assert.equal(error.status, 408);
  return true;
});
console.log('sdk error checks passed');
