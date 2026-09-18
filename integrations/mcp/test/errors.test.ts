import assert from 'node:assert/strict';
import { SandboxError } from '@haedes/sdk';
import { McpError, errorResult, toMcpError } from '../src/errors.js';

const error = new McpError('invalid_tool_arguments', 'The command is too long.', { maxLength: 32768 });
assert.equal(error.code, 'invalid_tool_arguments');
assert.deepEqual(errorResult(error), {
  isError: true,
  content: [{ type: 'text', text: 'The command is too long.' }],
  structuredContent: { code: 'invalid_tool_arguments', message: 'The command is too long.', details: { maxLength: 32768 } },
});
const platformError = toMcpError(new SandboxError({ code: 'authentication_required', message: 'Authentication required.', status: 401, requestId: 'req_1', details: { retryable: false } }));
assert.deepEqual(errorResult(platformError).structuredContent, {
  code: 'platform_error',
  message: 'Authentication required.',
  details: { platformCode: 'authentication_required', status: 401, requestId: 'req_1', platformDetails: { retryable: false } },
});
console.log('MCP error checks passed');
