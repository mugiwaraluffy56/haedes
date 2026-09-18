import assert from 'node:assert/strict';
import { createMcpServer } from '../src/server.js';

const sandbox = { id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] };
const command = { id: 'cmd_1', sandboxId: 'sbx_1', command: 'printf hello', exitCode: 0, startedAt: '2026-01-01T00:00:00Z', timedOut: false };
const snapshot = { id: 'snp_1', sandboxId: 'sbx_1', state: 'available', objectKey: 'opaque', createdAt: '2026-01-01T00:00:00Z' };
const entry = { path: '/workspace/hello.txt', kind: 'file' };
const eventResult = { id: command.id, sandboxId: command.sandboxId, command: command.command, exitCode: 0, startedAt: command.startedAt, timedOut: false };

function sseResponse(): Response {
  return new Response('id: 1\nevent: started\ndata: {"type":"started","commandId":"cmd_1","at":"2026-01-01T00:00:00Z"}\n\nid: 2\nevent: completed\ndata: {"type":"completed","commandId":"cmd_1","result":' + JSON.stringify(eventResult) + ',"at":"2026-01-01T00:00:01Z"}\n\n', { headers: { 'Content-Type': 'text/event-stream' } });
}

const requests: Array<{ url: string; init: RequestInit }> = [];
const fetcher: typeof fetch = async (input, init = {}) => {
  const url = String(input);
  requests.push({ url, init });
  if (url.includes('/v1/sandboxes?') && !init.method) return Response.json({ items: [sandbox], page: { hasMore: false } });
  if (url.endsWith('/v1/sandboxes') && init.method === 'POST') return Response.json(sandbox, { status: 202 });
  if (url.endsWith('/events')) return sseResponse();
  if (url.endsWith('/commands') && init.method === 'POST') return Response.json(command, { status: 202 });
  if (url.includes('/files/content')) {
    if (init.method === 'PUT' || init.method === 'DELETE') return new Response(null, { status: 204 });
    return new Response(new TextEncoder().encode('hello'));
  }
  if (url.includes('/files?')) return Response.json({ path: '/workspace', entries: [entry] });
  if (url.endsWith('/snapshots')) return Response.json(snapshot, { status: 202 });
  if (url.endsWith('/restore')) return new Response(null, { status: 204 });
  if (url.endsWith('/sbx_1') && init.method === 'DELETE') return Response.json({ ...sandbox, state: 'stopping' });
  if (url.endsWith('/sbx_1')) return Response.json(sandbox);
  throw new Error(`unexpected request: ${url}`);
};

const server = createMcpServer({ apiUrl: 'http://localhost:8080', apiKey: 'server-only-key', fetch: fetcher });
const call = async (name: string, arguments_: Record<string, unknown>) => {
  const result = await server.callTool(name, arguments_);
  return result.structuredContent as Record<string, any>;
};

assert.equal((await call('sandbox_create', {})).sandbox.id, 'sbx_1');
assert.equal((await call('sandbox_get', { sandboxId: 'sbx_1' })).sandbox.state, 'running');
assert.equal((await call('sandbox_list', { limit: 10 })).items[0].id, 'sbx_1');
assert.equal((await call('sandbox_exec', { sandboxId: 'sbx_1', request: { command: 'printf hello', timeoutSeconds: 10 } })).commandId, 'cmd_1');
assert.deepEqual((await call('sandbox_exec_stream', { sandboxId: 'sbx_1', commandId: 'cmd_1', lastEventId: '1' })).events.map((event: { type: string }) => event.type), ['started', 'completed']);
assert.equal((await call('sandbox_read_file', { sandboxId: 'sbx_1', path: '/workspace/hello.txt' })).content, 'hello');
assert.equal((await call('sandbox_write_file', { sandboxId: 'sbx_1', path: '/workspace/hello.txt', content: 'hello' })).entry.path, '/workspace/hello.txt');
assert.equal((await call('sandbox_list_files', { sandboxId: 'sbx_1' })).entries[0].path, '/workspace/hello.txt');
assert.equal((await call('sandbox_delete_file', { sandboxId: 'sbx_1', path: '/workspace/hello.txt' })).deleted, true);
assert.equal((await call('sandbox_snapshot', { sandboxId: 'sbx_1', expiresInSeconds: 60 })).snapshot.id, 'snp_1');
assert.equal((await call('sandbox_restore', { snapshotId: 'snp_1' })).sandbox.id, 'sbx_1');
assert.equal((await call('sandbox_destroy', { sandboxId: 'sbx_1' })).sandbox.state, 'stopping');
assert.equal(new Headers(requests.find((request) => request.url.endsWith('/events'))?.init.headers).get('Last-Event-ID'), '1');
assert.equal(JSON.stringify(requests).includes('server-only-key'), false);
console.log('MCP SDK tool checks passed');
