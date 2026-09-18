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
const contract = JSON.parse(await readFile(new URL('../src/tool-contract.schema.json', import.meta.url), 'utf8')) as { tools: Array<{ name: string }> };
assert.deepEqual(contract.tools.map((tool) => tool.name), toolDefinitions.map((tool) => tool.name));
assert.doesNotMatch(JSON.stringify(contract), /aws|ecs|fargate|dynamodb|s3|cloudwatch/i);
const registry = createToolRegistry({
  sandbox_get: async (arguments_) => structuredResult({ sandboxId: arguments_.sandboxId, state: 'running' }),
});
const result = await registry.call('sandbox_get', { sandboxId: 'sbx_1' }, { client: {} as SandboxClient });
assert.deepEqual(result.structuredContent, { sandboxId: 'sbx_1', state: 'running' });
console.log('MCP tool contract checks passed');
