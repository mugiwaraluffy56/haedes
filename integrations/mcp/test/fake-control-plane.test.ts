import assert from 'node:assert/strict';
import { createMcpServer } from '../src/server.js';

type SandboxRecord = { id: string; owner: string; state: string; config: Record<string, unknown>; snapshotIds: string[] };

class FakeControlPlane {
  readonly requests: Array<{ url: string; init: RequestInit }> = [];
  private nextSandbox = 1;
  private readonly sandboxes = new Map<string, SandboxRecord>();
  private readonly files = new Map<string, string>();
  private readonly snapshots = new Map<string, { id: string; sandboxId: string; state: string }>();

  fetch: typeof fetch = async (input, init = {}) => {
    const url = new URL(String(input));
    const request = { url: url.toString(), init };
    this.requests.push(request);
    const owner = new Headers(init.headers).get('Authorization')?.replace(/^Bearer /, '');
    if (!owner) return this.error(401, 'authentication_required', 'Authentication required.');

    if (url.pathname === '/v1/sandboxes' && init.method === 'POST') {
      const body = JSON.parse(String(init.body)) as { config: Record<string, unknown> };
      const id = `sbx_${this.nextSandbox++}`;
      const sandbox = { id, owner, state: 'running', config: body.config, snapshotIds: [] };
      this.sandboxes.set(id, sandbox);
      return Response.json(sandbox, { status: 202 });
    }
    if (url.pathname === '/v1/sandboxes' && !init.method) {
      return Response.json({ items: [...this.sandboxes.values()].filter((sandbox) => sandbox.owner === owner), page: { hasMore: false } });
    }
    const sandboxMatch = url.pathname.match(/^\/v1\/sandboxes\/([^/]+)$/);
    if (sandboxMatch) {
      const sandbox = this.sandboxes.get(decodeURIComponent(sandboxMatch[1]));
      if (!sandbox || sandbox.owner !== owner) return this.error(404, 'sandbox_not_found', 'Sandbox not found.');
      if (init.method === 'DELETE') {
        sandbox.state = 'destroyed';
        return Response.json(sandbox, { status: 202 });
      }
      return Response.json(sandbox);
    }
    const commandMatch = url.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/commands$/);
    if (commandMatch && init.method === 'POST') {
      const sandbox = this.requireSandbox(decodeURIComponent(commandMatch[1]), owner);
      const body = JSON.parse(String(init.body)) as { command: string };
      const command = { id: 'cmd_1', sandboxId: sandbox.id, command: body.command, exitCode: 0, startedAt: '2026-01-01T00:00:00Z', timedOut: false };
      return Response.json(command, { status: 202 });
    }
    if (url.pathname.endsWith('/events')) {
      const commandId = url.pathname.split('/').at(-2);
      const command = { id: commandId, sandboxId: 'sbx_1', command: 'printf ok', exitCode: 0, startedAt: '2026-01-01T00:00:00Z', timedOut: false };
      const body = `id: 1\nevent: started\ndata: {"type":"started","commandId":"${commandId}","at":"2026-01-01T00:00:00Z"}\n\nid: 2\nevent: completed\ndata: ${JSON.stringify({ type: 'completed', commandId, result: command, at: '2026-01-01T00:00:01Z' })}\n\n`;
      return new Response(body, { headers: { 'Content-Type': 'text/event-stream' } });
    }
    const fileMatch = url.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/files\/content$/);
    if (fileMatch) {
      const sandbox = this.requireSandbox(decodeURIComponent(fileMatch[1]), owner);
      const path = url.searchParams.get('path') ?? '';
      const key = `${sandbox.id}:${path}`;
      if (init.method === 'PUT') {
        this.files.set(key, await new Response(init.body as BodyInit).text());
        return new Response(null, { status: 204 });
      }
      if (init.method === 'DELETE') {
        this.files.delete(key);
        return new Response(null, { status: 204 });
      }
      return new Response(this.files.get(key) ?? '');
    }
    const listFilesMatch = url.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/files$/);
    if (listFilesMatch) {
      const sandbox = this.requireSandbox(decodeURIComponent(listFilesMatch[1]), owner);
      const path = url.searchParams.get('path') ?? '/workspace';
      const entries = [...this.files.keys()]
        .filter((key) => key.startsWith(`${sandbox.id}:${path}/`) || key === `${sandbox.id}:${path}`)
        .map((key) => ({ path: key.slice(sandbox.id.length + 1), kind: 'file' as const }));
      return Response.json({ path, entries });
    }
    const snapshotMatch = url.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/snapshots$/);
    if (snapshotMatch && init.method === 'POST') {
      const sandbox = this.requireSandbox(decodeURIComponent(snapshotMatch[1]), owner);
      const snapshot = { id: `snp_${this.snapshots.size + 1}`, sandboxId: sandbox.id, state: 'available' };
      this.snapshots.set(snapshot.id, snapshot);
      sandbox.snapshotIds.push(snapshot.id);
      return Response.json(snapshot, { status: 202 });
    }
    throw new Error(`unhandled fake control-plane request: ${init.method ?? 'GET'} ${url.pathname}`);
  };

  private requireSandbox(id: string, owner: string): SandboxRecord {
    const sandbox = this.sandboxes.get(id);
    if (!sandbox || sandbox.owner !== owner) throw new Error('sandbox_not_found');
    return sandbox;
  }

  private error(status: number, code: string, message: string): Response {
    return new Response(JSON.stringify({ error: { code, message, requestId: `req_${this.requests.length}`, details: {} } }), {
      status,
      headers: { 'Content-Type': 'application/json', 'X-Request-ID': `req_${this.requests.length}` },
    });
  }
}

const fake = new FakeControlPlane();
const ownerOne = createMcpServer({ apiUrl: 'http://fake.test', apiKey: 'owner-one', fetch: fake.fetch });
const ownerTwo = createMcpServer({ apiUrl: 'http://fake.test', apiKey: 'owner-two', fetch: fake.fetch });
const call = async (server: ReturnType<typeof createMcpServer>, name: string, arguments_: Record<string, unknown>) => {
  const response = await server.handle({ jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name, arguments: arguments_ } });
  return response?.result as { isError?: boolean; structuredContent?: Record<string, any> };
};

const created = await call(ownerOne, 'sandbox_create', {});
const sandboxId = created.structuredContent?.sandbox.id as string;
assert.equal(sandboxId, 'sbx_1');
assert.equal((await call(ownerOne, 'sandbox_list', {})).structuredContent?.items.length, 1);
assert.equal((await call(ownerTwo, 'sandbox_list', {})).structuredContent?.items.length, 0);
assert.equal((await call(ownerTwo, 'sandbox_get', { sandboxId })).structuredContent?.details.platformCode, 'sandbox_not_found');
assert.equal((await call(ownerOne, 'sandbox_exec', { sandboxId, request: { command: 'printf ok' } })).structuredContent?.commandId, 'cmd_1');
assert.equal((await call(ownerOne, 'sandbox_exec_stream', { sandboxId, commandId: 'cmd_1' })).structuredContent?.events.length, 2);
assert.equal((await call(ownerOne, 'sandbox_write_file', { sandboxId, path: '/workspace/answer.txt', content: '42' })).structuredContent?.entry.path, '/workspace/answer.txt');
assert.equal((await call(ownerOne, 'sandbox_read_file', { sandboxId, path: '/workspace/answer.txt' })).structuredContent?.content, '42');
assert.equal((await call(ownerOne, 'sandbox_list_files', { sandboxId })).structuredContent?.entries[0].path, '/workspace/answer.txt');
assert.equal((await call(ownerOne, 'sandbox_delete_file', { sandboxId, path: '/workspace/answer.txt' })).structuredContent?.deleted, true);
const snapshotId = (await call(ownerOne, 'sandbox_snapshot', { sandboxId })).structuredContent?.snapshot.id as string;
assert.equal((await call(ownerOne, 'sandbox_restore', { snapshotId })).structuredContent?.sandbox.state, 'running');
assert.equal((await call(ownerOne, 'sandbox_destroy', { sandboxId })).structuredContent?.sandbox.state, 'destroyed');

const requestsBeforeForbiddenInputs = fake.requests.length;
for (const key of ['awsAccessKeyId', 'awsSecretAccessKey', 'runtimeToken', 'taskArn', 'lifecycleState']) {
  const failure = await call(ownerOne, 'sandbox_create', { config: { [key]: 'secret-value' } });
  assert.equal(failure.isError, true, key);
  assert.equal(failure.structuredContent?.code, 'invalid_tool_arguments', key);
}
assert.equal(fake.requests.length, requestsBeforeForbiddenInputs);
assert.equal(JSON.stringify(fake.requests).includes('secret-value'), false);
console.log('MCP fake control-plane checks passed');
