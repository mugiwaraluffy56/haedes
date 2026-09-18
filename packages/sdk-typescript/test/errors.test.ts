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

const commandErrorClient = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  fetch: async (input) => {
    const url = String(input);
    if (url.endsWith('/v1/sandboxes/sbx_1')) {
      return Response.json({ id: 'sbx_1', state: 'provisioning', config: {}, snapshotIds: [] });
    }
    if (url.endsWith('/commands')) {
      return new Response(JSON.stringify({ error: { code: 'sandbox_not_running', message: 'Sandbox is provisioning.', requestId: 'req_2', details: { state: 'provisioning' } } }), {
        status: 409,
        headers: { 'Content-Type': 'application/json', 'X-Request-ID': 'req_2' },
      });
    }
    throw new Error(`unexpected request: ${url}`);
  },
});
await assert.rejects((await commandErrorClient.sandboxes.get('sbx_1')).exec('printf hello'), (error: unknown) => {
  assert.ok(error instanceof SandboxError);
  assert.equal(error.code, 'sandbox_not_running');
  assert.equal(error.requestId, 'req_2');
  assert.equal(error.status, 409);
  return true;
});

const fileErrorClient = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  fetch: async (input) => {
    const url = String(input);
    if (url.endsWith('/v1/sandboxes/sbx_1')) {
      return Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] });
    }
    if (url.includes('/files/content')) {
      return new Response(JSON.stringify({ error: { code: 'file_not_found', message: 'Missing file.', requestId: 'req_3', details: {} } }), {
        status: 404,
        headers: { 'Content-Type': 'application/json', 'X-Request-ID': 'req_3' },
      });
    }
    if (url.endsWith('/snapshots')) {
      return new Response(JSON.stringify({ error: { code: 'snapshot_conflict', message: 'Snapshot is unavailable.', requestId: 'req_4', details: {} } }), {
        status: 409,
        headers: { 'Content-Type': 'application/json', 'X-Request-ID': 'req_4' },
      });
    }
    throw new Error(`unexpected request: ${url}`);
  },
});
const fileErrorSandbox = await fileErrorClient.sandboxes.get('sbx_1');
await assert.rejects(fileErrorSandbox.readFile('/workspace/missing.txt'), (error: unknown) => {
  assert.ok(error instanceof SandboxError);
  assert.equal(error.code, 'file_not_found');
  assert.equal(error.requestId, 'req_3');
  return true;
});
await assert.rejects(fileErrorSandbox.snapshot(), (error: unknown) => {
  assert.ok(error instanceof SandboxError);
  assert.equal(error.code, 'snapshot_conflict');
  assert.equal(error.requestId, 'req_4');
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
