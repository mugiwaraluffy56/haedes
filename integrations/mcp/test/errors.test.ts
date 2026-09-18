import assert from 'node:assert/strict';
import { McpError, errorResult } from '../src/errors.js';

const error = new McpError('invalid_tool_arguments', 'The command is too long.', { maxLength: 32768 });
assert.equal(error.code, 'invalid_tool_arguments');
assert.deepEqual(errorResult(error), {
  isError: true,
  content: [{ type: 'text', text: 'The command is too long.' }],
  structuredContent: { code: 'invalid_tool_arguments', message: 'The command is too long.', details: { maxLength: 32768 } },
});
console.log('MCP error checks passed');
