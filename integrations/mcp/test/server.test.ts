import assert from 'node:assert/strict';
import { McpError } from '../src/errors.js';
import { configFromEnvironment, createMcpServer } from '../src/server.js';

assert.deepEqual(configFromEnvironment({ HAEDES_API_URL: 'http://localhost:8080', HAEDES_API_KEY: 'server-only-key' }), {
  apiUrl: 'http://localhost:8080', apiKey: 'server-only-key',
});
assert.throws(() => configFromEnvironment({ HAEDES_API_URL: 'http://localhost:8080' }), (error: unknown) => error instanceof McpError && error.code === 'configuration_invalid');

const server = createMcpServer({
  apiUrl: 'http://localhost:8080',
  apiKey: 'server-only-key',
  fetch: async () => Response.json({ id: 'sbx_1', state: 'running', config: {}, snapshotIds: [] }),
});
assert.equal(server.listTools().length, 12);
assert.ok(!JSON.stringify(server.listTools()).includes('server-only-key'));
const initialize = await server.handle({ jsonrpc: '2.0', id: 1, method: 'initialize' });
assert.equal((initialize?.result as { serverInfo: { name: string } }).serverInfo.name, 'haedes');
const list = await server.handle({ jsonrpc: '2.0', id: 2, method: 'tools/list' });
assert.equal((list?.result as { tools: unknown[] }).tools.length, 12);
const toolCall = await server.handle({ jsonrpc: '2.0', id: 3, method: 'tools/call', params: { name: 'sandbox_get', arguments: { sandboxId: 'sbx_1' } } });
assert.equal((toolCall?.result as { isError?: boolean; structuredContent: { sandbox: { id: string } } }).structuredContent.sandbox.id, 'sbx_1');
const authServer = createMcpServer({
  apiUrl: 'http://localhost:8080',
  apiKey: 'server-only-key',
  fetch: async () => new Response(JSON.stringify({ error: { code: 'authentication_required', message: 'Authentication required.', requestId: 'req_1', details: {} } }), { status: 401, headers: { 'Content-Type': 'application/json', 'X-Request-ID': 'req_1' } }),
});
const authFailure = await authServer.handle({ jsonrpc: '2.0', id: 4, method: 'tools/call', params: { name: 'sandbox_get', arguments: { sandboxId: 'sbx_1' } } });
assert.deepEqual((authFailure?.result as { structuredContent: unknown }).structuredContent, {
  code: 'platform_error', message: 'Authentication required.', details: { platformCode: 'authentication_required', status: 401, requestId: 'req_1' },
});
assert.equal(JSON.stringify(authFailure).includes('server-only-key'), false);
console.log('MCP server bootstrap checks passed');
