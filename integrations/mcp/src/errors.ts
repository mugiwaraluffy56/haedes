export type McpErrorCode =
  | 'configuration_invalid'
  | 'invalid_tool_arguments'
  | 'tool_not_found'
  | 'tool_not_implemented'
  | 'protocol_invalid';

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
