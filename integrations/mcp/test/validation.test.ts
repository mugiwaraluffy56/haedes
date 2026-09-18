import assert from 'node:assert/strict';
import { McpError } from '../src/errors.js';
import { validateToolArguments } from '../src/validation.js';

assert.deepEqual(validateToolArguments('sandbox_exec', { sandboxId: 'sbx_1', request: { command: 'npm test', timeoutSeconds: 30 } }), {
  sandboxId: 'sbx_1', request: { command: 'npm test', timeoutSeconds: 30 },
});
for (const input of [
  { sandboxId: 'sbx_1', request: { command: '' } },
  { sandboxId: 'sbx_1', request: { command: 'npm test', timeoutSeconds: 0 } },
  { sandboxId: 'sbx_1', request: { command: 'npm test' }, apiKey: 'must-not-be-tool-input' },
  { sandboxId: 'sbx_1', request: { command: 'npm test' }, awsAccessKeyId: 'must-not-be-tool-input' },
]) {
  assert.throws(() => validateToolArguments('sandbox_exec', input), (error: unknown) => error instanceof McpError && error.code === 'invalid_tool_arguments');
}
assert.throws(() => validateToolArguments('unknown_tool', {}), (error: unknown) => error instanceof McpError && error.code === 'tool_not_found');
console.log('MCP validation checks passed');
