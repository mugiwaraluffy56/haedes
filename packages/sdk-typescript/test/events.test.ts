import assert from 'node:assert/strict';
import { SandboxClient } from '../src/index.js';

let eventRequests = 0;
const lastEventIds: Array<string | null> = [];
const client = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  fetch: async (input, init = {}) => {
    const url = String(input);
    if (url.endsWith('/commands')) {
      return Response.json({ id: 'cmd_1', sandboxId: 'sbx_1', command: 'printf output', exitCode: 0, startedAt: new Date().toISOString(), timedOut: false }, { status: 202 });
    }
    if (url.endsWith('/v1/sandboxes/sbx_1')) {
      return Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] });
    }
    if (url.endsWith('/events')) {
      eventRequests += 1;
      lastEventIds.push(new Headers(init.headers).get('Last-Event-ID'));
      if (eventRequests === 1) {
        return new Response('id: 1\nevent: started\ndata: {"type":"started","commandId":"cmd_1","at":"2026-01-01T00:00:00Z"}\n\nid: 2\nevent: stdout\ndata: {\ndata:   "type": "stdout",\ndata:   "commandId": "cmd_1",\ndata:   "data": "line 1\\nline 2",\ndata:   "at": "2026-01-01T00:00:00Z"\ndata: }\n\n');
      }
      return new Response('id: 3\nevent: completed\ndata: {"type":"completed","commandId":"cmd_1","result":{"id":"cmd_1","sandboxId":"sbx_1","command":"printf output","exitCode":0,"startedAt":"2026-01-01T00:00:00Z","timedOut":false},"at":"2026-01-01T00:00:01Z"}\n\n');
    }
    throw new Error(`unexpected request: ${url}`);
  },
});

const events = [];
for await (const event of (await client.sandboxes.get('sbx_1')).execStream('printf output')) events.push(event);
assert.deepEqual(events.map((event) => event.type), ['started', 'stdout', 'completed']);
assert.deepEqual(lastEventIds, [null, '2']);
assert.equal(events.at(1)?.type, 'stdout');
assert.equal((events[1] as { data: string }).data, 'line 1\nline 2');
console.log('sdk event checks passed');
