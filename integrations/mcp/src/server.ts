import { createInterface, type ReadLine } from 'node:readline';
import { errorResult, McpError, toMcpError } from './errors.js';
import { createPlatformClient, type PlatformClient } from './platform-client.js';
import { createSdkToolHandlers, createToolRegistry, type ToolHandler, type ToolName, type ToolResult } from './tools.js';

export interface McpServerConfig {
  apiUrl: string;
  apiKey: string;
  fetch?: typeof fetch;
}

export function configFromEnvironment(environment: NodeJS.ProcessEnv = process.env): McpServerConfig {
  const apiUrl = environment.HAEDES_API_URL;
  const apiKey = environment.HAEDES_API_KEY;
  if (!apiUrl || !apiKey || apiUrl.trim() === '' || apiKey.trim() === '') {
    throw new McpError('configuration_invalid', 'HAEDES_API_URL and HAEDES_API_KEY must be configured on the server.');
  }
  return { apiUrl, apiKey };
}

export interface JsonRpcRequest {
  jsonrpc: '2.0';
  id?: string | number;
  method: string;
  params?: Record<string, unknown>;
}

export interface JsonRpcResponse {
  jsonrpc: '2.0';
  id?: string | number | null;
  result?: unknown;
  error?: { code: number; message: string; data?: unknown };
}

export class McpServer {
  private readonly registry;

  constructor(private readonly platform: PlatformClient, handlers: Partial<Record<ToolName, ToolHandler>> = {}) {
    this.registry = createToolRegistry(handlers);
  }

  listTools() {
    return this.registry.definitions;
  }

  async callTool(name: string, arguments_: unknown): Promise<ToolResult> {
    try {
      return await this.registry.call(name, arguments_, { client: this.platform.sdk });
    } catch (error) {
      throw toMcpError(error);
    }
  }

  async handle(request: JsonRpcRequest): Promise<JsonRpcResponse | undefined> {
    if (request.method === 'notifications/initialized') return undefined;
    if (request.method === 'initialize') {
      return { jsonrpc: '2.0', id: request.id, result: { protocolVersion: '2024-11-05', capabilities: { tools: {} }, serverInfo: { name: 'haedes', version: '0.1.0' } } };
    }
    if (request.method === 'tools/list') return { jsonrpc: '2.0', id: request.id, result: { tools: this.listTools() } };
    if (request.method === 'tools/call') {
      const name = request.params?.name;
      if (typeof name !== 'string') return this.protocolError(request.id, 'tools/call requires a tool name.');
      try {
        const result = await this.callTool(name, request.params?.arguments ?? {});
        return { jsonrpc: '2.0', id: request.id, result };
      } catch (error) {
        return { jsonrpc: '2.0', id: request.id, result: errorResult(toMcpError(error)) };
      }
    }
    return this.protocolError(request.id, `Unsupported MCP method ${request.method}.`);
  }

  private protocolError(id: string | number | undefined, message: string): JsonRpcResponse {
    return { jsonrpc: '2.0', id: id ?? null, error: { code: -32602, message } };
  }
}

export function createMcpServer(config: McpServerConfig, handlers: Partial<Record<ToolName, ToolHandler>> = {}): McpServer {
  return new McpServer(createPlatformClient({ baseUrl: config.apiUrl, apiKey: config.apiKey, ...(config.fetch ? { fetch: config.fetch } : {}) }), { ...createSdkToolHandlers(), ...handlers });
}

export async function startStdioServer(server: McpServer, input: NodeJS.ReadableStream = process.stdin, output: NodeJS.WritableStream = process.stdout): Promise<void> {
  const lines: ReadLine = createInterface({ input, crlfDelay: Infinity });
  for await (const line of lines) {
    if (line.trim() === '') continue;
    let request: JsonRpcRequest;
    try {
      request = JSON.parse(line) as JsonRpcRequest;
      const response = await server.handle(request);
      if (response) output.write(`${JSON.stringify(response)}\n`);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Invalid MCP request.';
      output.write(`${JSON.stringify({ jsonrpc: '2.0', id: null, error: { code: -32600, message } })}\n`);
    }
  }
}

if (process.argv[1] && process.argv[1].endsWith('/server.js')) {
  try {
    await startStdioServer(createMcpServer(configFromEnvironment()));
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : 'Unable to start MCP server.'}\n`);
    process.exitCode = 1;
  }
}
