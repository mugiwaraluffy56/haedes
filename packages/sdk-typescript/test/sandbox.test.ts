import assert from 'node:assert/strict';
import { SandboxClient } from '../src/index.js';

const calls: Array<{ url: string; init: RequestInit }> = [];
const client = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  fetch: async (input, init = {}) => {
    const url = String(input);
    calls.push({ url, init });
    if (url.includes('/files/content')) {
      if (init.method === 'PUT') return new Response(null, { status: 204 });
      if (init.method === 'DELETE') return new Response(null, { status: 204 });
      return new Response(new TextEncoder().encode('hello'));
    }
    if (url.includes('/files?')) return Response.json({ path: '/workspace', entries: [{ path: '/workspace/a.txt', kind: 'file' }] });
    if (url.endsWith('/snapshots')) return Response.json({ id: 'snp_1', state: 'available' }, { status: 202 });
    if (url.endsWith('/restore')) return new Response(null, { status: 204 });
    if (url.endsWith('/commands')) return Response.json({ id: 'cmd_1', sandboxId: 'sbx_1', command: 'printf hello', exitCode: 0, startedAt: new Date().toISOString(), timedOut: false }, { status: 202 });
    if (url.endsWith('/sbx_1')) return Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] });
    throw new Error(`unexpected request: ${url}`);
  },
});

const sandbox = await client.sandboxes.get('sbx_1');
const command = await sandbox.exec('printf hello', {
  cwd: '/workspace',
  environment: { TEST_MODE: 'sdk' },
  timeoutSeconds: 15,
});
assert.equal(command.id, 'cmd_1');
const commandCall = calls.find((call) => call.url.endsWith('/commands'));
assert.ok(commandCall);
assert.deepEqual(JSON.parse(String(commandCall.init.body)), {
  command: 'printf hello',
  cwd: '/workspace',
  environment: { TEST_MODE: 'sdk' },
  timeoutSeconds: 15,
});
assert.equal(new Headers(commandCall.init.headers).get('Authorization'), 'Bearer key');

await assert.rejects(sandbox.exec(''), /command must not be empty/);
await assert.rejects(sandbox.exec('printf hello', { timeoutSeconds: 901 }), /timeoutSeconds must be an integer/);
await assert.rejects(sandbox.exec('x'.repeat(16 * 1024 + 1)), /command must not exceed/);
await assert.rejects(
  sandbox.exec('printf hello', { environment: Object.fromEntries(Array.from({ length: 33 }, (_, index) => [`KEY_${index}`, 'value'])) }),
  /environment must not contain more than 32 entries/,
);

await sandbox.writeFile('/workspace/a.txt', 'hello');
assert.equal(await sandbox.readFile('/workspace/a.txt'), 'hello');
assert.equal((await sandbox.listFiles()).length, 1);
await sandbox.deleteFile('/workspace/a.txt');
assert.equal((await sandbox.snapshot({ expiresInSeconds: 60 })).id, 'snp_1');
await sandbox.restore('snp_1');

assert.equal(calls.some((call) => call.url.includes('path=%2Fworkspace%2Fa.txt')), true);
assert.equal(calls.filter((call) => call.init.method === 'POST').length, 3);
console.log('sandbox handle checks passed');
