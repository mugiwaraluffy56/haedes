import assert from 'node:assert/strict';
import { SandboxClient } from '../../packages/sdk-typescript/src/index.js';
import { createMcpServer } from '../../integrations/mcp/src/server.js';
import { createDashboardApi } from '../../apps/dashboard/src/lib/api-client.js';

type SandboxRecord = { id: string; state: string; config: Record<string, unknown>; snapshotIds: string[] };

class JourneyApi {
  private sandboxNumber = 0;
  private commandNumber = 0;
  private snapshotNumber = 0;
  private readonly sandboxes = new Map<string, SandboxRecord>();
  private readonly files = new Map<string, Map<string, string>>();
  private readonly snapshots = new Map<string, { sandboxId: string; files: Map<string, string> }>();
  readonly authorizationHeaders: string[] = [];

  fetch: typeof fetch = async (input, init = {}) => {
    const request = new URL(String(input));
    const authorization = new Headers(init.headers).get('Authorization');
    if (authorization) this.authorizationHeaders.push(authorization);
    if (authorization !== 'Bearer journey-key') return this.error(401, 'authentication_required', 'Authentication required.');

    if (request.pathname === '/v1/sandboxes' && init.method === 'POST') {
      const body = JSON.parse(String(init.body)) as { config: Record<string, unknown> };
      const id = `sbx_journey_${++this.sandboxNumber}`;
      const config = body.config;
      const sandbox: SandboxRecord = { id, state: 'running', config, snapshotIds: [] };
      this.sandboxes.set(id, sandbox);
      const files = new Map<string, string>();
      const snapshotId = typeof config.snapshotId === 'string' ? config.snapshotId : undefined;
      if (snapshotId) {
        const snapshot = this.snapshots.get(snapshotId);
        if (snapshot) {
          for (const [path, content] of snapshot.files) files.set(path, content);
          sandbox.snapshotIds.push(snapshotId);
        }
      }
      this.files.set(id, files);
      return Response.json(sandbox, { status: 202 });
    }
    if (request.pathname === '/v1/sandboxes' && !init.method) {
      return Response.json({ items: [...this.sandboxes.values()], page: { hasMore: false } });
    }

    const sandboxMatch = request.pathname.match(/^\/v1\/sandboxes\/([^/]+)$/);
    if (sandboxMatch) {
      const sandbox = this.sandboxes.get(decodeURIComponent(sandboxMatch[1]));
      if (!sandbox) return this.error(404, 'sandbox_not_found', 'Sandbox not found.');
      if (init.method === 'DELETE') sandbox.state = 'destroyed';
      return Response.json(sandbox, { status: init.method === 'DELETE' ? 202 : 200 });
    }

    const commandMatch = request.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/commands$/);
    if (commandMatch && init.method === 'POST') {
      const sandboxId = decodeURIComponent(commandMatch[1]);
      const body = JSON.parse(String(init.body)) as { command: string };
      const id = `cmd_journey_${++this.commandNumber}`;
      return Response.json({ id, sandboxId, command: body.command, exitCode: 0, startedAt: '2026-01-01T00:00:00Z', timedOut: false }, { status: 202 });
    }
    if (request.pathname.endsWith('/events')) {
      const commandId = request.pathname.split('/').at(-2);
      const result = { id: commandId, sandboxId: 'sbx_journey_1', command: 'printf journey', exitCode: 0, startedAt: '2026-01-01T00:00:00Z', timedOut: false };
      const body = `id: 1\nevent: started\ndata: {"type":"started","commandId":"${commandId}","at":"2026-01-01T00:00:00Z"}\n\nid: 2\nevent: completed\ndata: ${JSON.stringify({ type: 'completed', commandId, result, at: '2026-01-01T00:00:01Z' })}\n\n`;
      return new Response(body, { headers: { 'Content-Type': 'text/event-stream' } });
    }

    const fileMatch = request.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/files\/content$/);
    if (fileMatch) {
      const sandboxId = decodeURIComponent(fileMatch[1]);
      const path = request.searchParams.get('path') ?? '';
      const files = this.files.get(sandboxId);
      if (!files) return this.error(404, 'sandbox_not_found', 'Sandbox not found.');
      if (init.method === 'PUT') {
        files.set(path, await new Response(init.body as BodyInit).text());
        return new Response(null, { status: 204 });
      }
      if (init.method === 'DELETE') {
        files.delete(path);
        return new Response(null, { status: 204 });
      }
      return new Response(files.get(path) ?? '', { status: files.has(path) ? 200 : 404 });
    }
    const filesMatch = request.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/files$/);
    if (filesMatch) {
      const sandboxId = decodeURIComponent(filesMatch[1]);
      const path = request.searchParams.get('path') ?? '/workspace';
      const entries = [...(this.files.get(sandboxId)?.keys() ?? [])]
        .filter((candidate) => candidate === path || candidate.startsWith(`${path}/`))
        .map((candidate) => ({ path: candidate, kind: 'file' as const }));
      return Response.json({ path, entries });
    }
    const snapshotMatch = request.pathname.match(/^\/v1\/sandboxes\/([^/]+)\/snapshots$/);
    if (snapshotMatch && init.method === 'POST') {
      const sandboxId = decodeURIComponent(snapshotMatch[1]);
      const id = `snp_journey_${++this.snapshotNumber}`;
      this.snapshots.set(id, { sandboxId, files: new Map(this.files.get(sandboxId) ?? []) });
      this.sandboxes.get(sandboxId)?.snapshotIds.push(id);
      return Response.json({ id, sandboxId, state: 'available', objectKey: 'fake', createdAt: '2026-01-01T00:00:00Z' }, { status: 202 });
    }
    throw new Error(`Unhandled journey request: ${init.method ?? 'GET'} ${request.pathname}`);
  };

  private error(status: number, code: string, message: string): Response {
    return Response.json({ error: { code, message, requestId: 'req_journey', details: {} } }, { status });
  }
}

async function main(): Promise<void> {
  const api = new JourneyApi();
  const sdk = new SandboxClient({ baseUrl: 'http://journey.test', apiKey: 'journey-key', fetch: api.fetch });
  const sandbox = await sdk.sandboxes.create({ image: 'journey' });
  const command = await sandbox.exec('printf journey');
  const events = [];
  for await (const event of sandbox.execStream('printf journey')) events.push(event);
  assert.equal(command.exitCode, 0);
  assert.deepEqual(events.map((event) => event.type), ['started', 'completed']);
  await sandbox.writeFile('/workspace/journey.txt', 'before snapshot');
  assert.equal(await sandbox.readFile('/workspace/journey.txt'), 'before snapshot');
  assert.equal((await sandbox.listFiles()).length, 1);
  const snapshot = await sandbox.snapshot();
  assert.equal(snapshot.state, 'available');
  const destroyed = await sandbox.destroy();
  assert.equal(destroyed.state, 'destroyed');
  const restored = await sdk.sandboxes.create({ snapshotId: snapshot.id });
  assert.equal(await restored.readFile('/workspace/journey.txt'), 'before snapshot');

  const dashboard = createDashboardApi({ baseUrl: 'http://journey.test', apiKey: 'journey-key', fetch: api.fetch });
  assert.ok((await dashboard.listSandboxes()).items.some((item) => item.id === restored.id));
  assert.equal((await dashboard.getSandbox(restored.id)).id, restored.id);
  const dashboardSnapshot = await dashboard.createSnapshot(restored.id);
  assert.equal(dashboardSnapshot.state, 'available');
  const dashboardRestored = await dashboard.restoreSnapshot(dashboardSnapshot.id);
  assert.notEqual(dashboardRestored.id, restored.id);
  assert.equal((await dashboard.destroySandbox(dashboardRestored.id)).state, 'destroyed');

  const mcp = createMcpServer({ apiUrl: 'http://journey.test', apiKey: 'journey-key', fetch: api.fetch });
  const mcpList = await mcp.callTool('sandbox_list', {});
  assert.ok((mcpList.structuredContent as { items: unknown[] }).items.length >= 3);
  assert.ok(api.authorizationHeaders.length > 0);
  assert.ok(api.authorizationHeaders.every((value) => value === 'Bearer journey-key'));
  console.log('SDK, MCP, and dashboard journey checks passed');
}

main().catch((error: unknown) => {
  console.error(error);
  process.exitCode = 1;
});
