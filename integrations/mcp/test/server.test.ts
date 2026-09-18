import assert from 'node:assert/strict';
import { McpError } from '../src/errors.js';
import { configFromEnvironment, createMcpServer } from '../src/server.js';

assert.deepEqual(configFromEnvironment({ HAEDES_API_URL: 'http://localhost:8080', HAEDES_API_KEY: 'server-only-key' }), {
  apiUrl: 'http://localhost:8080', apiKey: 'server-only-key',
});
assert.throws(() => configFromEnvironment({ HAEDES_API_URL: 'http://localhost:8080' }), (error: unknown) => error instanceof McpError && error.code === 'configuration_invalid');

const server = createMcpServer({ apiUrl: 'http://localhost:8080', apiKey: 'server-only-key' });
assert.equal(server.listTools().length, 12);
assert.ok(!JSON.stringify(server.listTools()).includes('server-only-key'));
const initialize = await server.handle({ jsonrpc: '2.0', id: 1, method: 'initialize' });
assert.equal((initialize?.result as { serverInfo: { name: string } }).serverInfo.name, 'haedes');
const list = await server.handle({ jsonrpc: '2.0', id: 2, method: 'tools/list' });
assert.equal((list?.result as { tools: unknown[] }).tools.length, 12);
const missingImplementation = await server.handle({ jsonrpc: '2.0', id: 3, method: 'tools/call', params: { name: 'sandbox_get', arguments: { sandboxId: 'sbx_1' } } });
assert.equal((missingImplementation?.result as { isError: boolean }).isError, true);
console.log('MCP server bootstrap checks passed');
