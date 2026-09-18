import assert from 'node:assert/strict';
import { SandboxClient } from '../src/index.js';

function sseResponse(chunks: string[]): Response {
  const encoder = new TextEncoder();
  return new Response(new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(encoder.encode(chunk));
      controller.close();
    },
  }), { headers: { 'Content-Type': 'text/event-stream' } });
}

let eventRequests = 0;
const lastEventIds: Array<string | null> = [];
const eventHeaders: Headers[] = [];
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
      eventHeaders.push(new Headers(init.headers));
      if (eventRequests === 1) {
        return sseResponse([
          'id: 1\nevent: started\ndata: {"type":"started","commandId":"cmd_1",',
          '"at":"2026-01-01T00:00:00Z"}\n\nid: 2\nevent: stdout\ndata: {\n',
          'data:   "type": "stdout",\ndata:   "commandId": "cmd_1",\n',
          'data:   "data": "line 1\\nline 2",\ndata:   "at": "2026-01-01T00:00:00Z"\n',
          'data: }\n\n',
        ]);
      }
      return sseResponse(['id: 3\r\nevent: completed\r\ndata: {"type":"completed","commandId":"cmd_1","result":{"id":"cmd_1","sandboxId":"sbx_1","command":"printf output","exitCode":0,"startedAt":"2026-01-01T00:00:00Z","timedOut":false},"at":"2026-01-01T00:00:01Z"}\r\n\r\n']);
    }
    throw new Error(`unexpected request: ${url}`);
  },
});

const events = [];
for await (const event of (await client.sandboxes.get('sbx_1')).execStream('printf output')) events.push(event);
assert.deepEqual(events.map((event) => event.type), ['started', 'stdout', 'completed']);
assert.deepEqual(lastEventIds, [null, '2']);
assert.equal(eventHeaders[0].get('Accept'), 'text/event-stream');
assert.equal(eventHeaders[0].get('Authorization'), 'Bearer key');
assert.equal(eventHeaders[1].get('Last-Event-ID'), '2');
assert.equal(events.at(1)?.type, 'stdout');
assert.equal((events[1] as { data: string }).data, 'line 1\nline 2');

const failedClient = new SandboxClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'key',
  fetch: async (input, init = {}) => {
    const url = String(input);
    if (url.endsWith('/commands')) {
      return Response.json({ id: 'cmd_2', sandboxId: 'sbx_1', command: 'false', exitCode: null, startedAt: '2026-01-01T00:00:00Z', timedOut: false }, { status: 202 });
    }
    if (url.endsWith('/v1/sandboxes/sbx_1')) {
      return Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] });
    }
    if (url.endsWith('/events')) {
      assert.equal(new Headers(init.headers).get('Last-Event-ID'), null);
      return sseResponse(['id: 1\nevent: failed\ndata: {"type":"failed","commandId":"cmd_2","code":"command_failed","message":"exit status 1","at":"2026-01-01T00:00:01Z"}\n\n']);
    }
    throw new Error(`unexpected request: ${url}`);
  },
});
const failedEvents = [];
for await (const event of (await failedClient.sandboxes.get('sbx_1')).execStream('false')) failedEvents.push(event);
assert.deepEqual(failedEvents, [{
  type: 'failed',
  commandId: 'cmd_2',
  code: 'command_failed',
  message: 'exit status 1',
  at: '2026-01-01T00:00:01Z',
}]);
console.log('sdk event checks passed');
