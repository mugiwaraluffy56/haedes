import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import type { SandboxClient } from '@haedes/sdk';
import { createToolRegistry, structuredResult, toolDefinitions } from '../src/tools.js';

assert.equal(toolDefinitions.length, 12);
assert.deepEqual(toolDefinitions.map((tool) => tool.name), [
  'sandbox_create', 'sandbox_get', 'sandbox_list', 'sandbox_exec', 'sandbox_exec_stream',
  'sandbox_read_file', 'sandbox_write_file', 'sandbox_list_files', 'sandbox_delete_file',
  'sandbox_snapshot', 'sandbox_restore', 'sandbox_destroy',
]);
const contract = JSON.parse(await readFile(new URL('../src/tool-contract.schema.json', import.meta.url), 'utf8')) as {
  $schema: string;
  $defs: Record<string, unknown>;
  tools: Array<{ name: string; inputSchema: Record<string, unknown>; resultSchema: Record<string, unknown> }>;
};
assert.equal(contract.$schema, 'https://json-schema.org/draft/2020-12/schema');
assert.ok(Object.keys(contract.$defs).length > 0);
assert.deepEqual(contract.tools.map((tool) => tool.name), toolDefinitions.map((tool) => tool.name));
assert.equal(new Set(contract.tools.map((tool) => tool.name)).size, contract.tools.length);
for (const tool of contract.tools) {
  assert.equal(typeof tool.inputSchema, 'object');
  assert.equal(typeof tool.resultSchema, 'object');
  assert.ok(tool.inputSchema.type === 'object' || typeof tool.inputSchema.$ref === 'string');
  assert.ok(tool.resultSchema.type === 'object' || typeof tool.resultSchema.$ref === 'string');
}
assert.doesNotMatch(JSON.stringify(contract), /aws|ecs|fargate|dynamodb|s3|cloudwatch/i);
const registry = createToolRegistry({
  sandbox_get: async (arguments_) => structuredResult({ sandboxId: arguments_.sandboxId, state: 'running' }),
});
const result = await registry.call('sandbox_get', { sandboxId: 'sbx_1' }, { client: {} as SandboxClient });
assert.deepEqual(result.structuredContent, { sandboxId: 'sbx_1', state: 'running' });
console.log('MCP tool contract checks passed');
