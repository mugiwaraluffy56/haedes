import { SandboxError } from '@haedes/sdk';

export type McpErrorCode =
  | 'configuration_invalid'
  | 'invalid_tool_arguments'
  | 'tool_not_found'
  | 'tool_not_implemented'
  | 'protocol_invalid'
  | 'platform_error';

export class McpError extends Error {
  constructor(readonly code: McpErrorCode, message: string, readonly details: Record<string, unknown> = {}) {
    super(message);
    this.name = 'McpError';
  }
}

export function errorResult(error: McpError): { isError: true; content: [{ type: 'text'; text: string }]; structuredContent: { code: string; message: string; details: Record<string, unknown> } } {
  return {
    isError: true,
    content: [{ type: 'text', text: error.message }],
    structuredContent: { code: error.code, message: error.message, details: error.details },
  };
}

export function toMcpError(error: unknown): McpError {
  if (error instanceof McpError) return error;
  if (error instanceof SandboxError) {
    return new McpError('platform_error', error.message, {
      platformCode: error.code,
      status: error.status,
      ...(error.requestId ? { requestId: error.requestId } : {}),
      ...(Object.keys(error.details).length > 0 ? { platformDetails: error.details } : {}),
    });
  }
  if (error instanceof Error) return new McpError('platform_error', error.message);
  return new McpError('platform_error', 'The sandbox platform request failed.');
}
