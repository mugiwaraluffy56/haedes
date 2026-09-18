import type { SandboxClient } from '@haedes/sdk';
import { McpError } from './errors.js';
import { validateToolArguments } from './validation.js';

export interface JsonSchema {
  type?: 'object' | 'array' | 'string' | 'integer' | 'number' | 'boolean' | 'null';
  properties?: Record<string, JsonSchema>;
  required?: string[];
  additionalProperties?: boolean | JsonSchema;
  items?: JsonSchema;
  enum?: readonly unknown[];
  const?: unknown;
  minLength?: number;
  maxLength?: number;
  minimum?: number;
  maximum?: number;
  maxProperties?: number;
  pattern?: string;
  contentEncoding?: string;
}

export type ToolName =
  | 'sandbox_create'
  | 'sandbox_get'
  | 'sandbox_list'
  | 'sandbox_exec'
  | 'sandbox_exec_stream'
  | 'sandbox_read_file'
  | 'sandbox_write_file'
  | 'sandbox_list_files'
  | 'sandbox_delete_file'
  | 'sandbox_snapshot'
  | 'sandbox_restore'
  | 'sandbox_destroy';

export interface ToolDefinition {
  name: ToolName;
  description: string;
  inputSchema: JsonSchema;
  resultSchema: JsonSchema;
}

export interface ToolContent {
  type: 'text';
  text: string;
}

export interface ToolResult<T = unknown> {
  content: [ToolContent];
  structuredContent?: T;
  isError?: boolean;
}

export interface ToolContext {
  readonly client: SandboxClient;
}

export type ToolHandler = (arguments_: Record<string, unknown>, context: ToolContext) => Promise<ToolResult>;

const idSchema: JsonSchema = { type: 'string', minLength: 1, maxLength: 128, pattern: '^[A-Za-z0-9._:-]+$' };
const workspacePathSchema: JsonSchema = { type: 'string', minLength: 1, maxLength: 4096, pattern: '^/workspace(?:/[^\\u0000-\\u001f]*)?$' };
const filePathSchema: JsonSchema = { type: 'string', minLength: 1, maxLength: 4096, pattern: '^/workspace/[^\\u0000-\\u001f]*$' };
const environmentSchema: JsonSchema = { type: 'object', maxProperties: 64, additionalProperties: { type: 'string', maxLength: 4096 } };
const repositorySchema: JsonSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['provider', 'url', 'path'],
  properties: {
    provider: { const: 'github' },
    url: { type: 'string', minLength: 1, maxLength: 2048 },
    ref: { type: 'string', minLength: 1, maxLength: 256 },
    path: workspacePathSchema,
  },
};
const sandboxConfigSchema: JsonSchema = {
  type: 'object',
  additionalProperties: false,
  properties: {
    image: { type: 'string', minLength: 1, maxLength: 256 },
    cpuMillis: { type: 'integer', minimum: 50, maximum: 4096 },
    memoryMiB: { type: 'integer', minimum: 128, maximum: 16384 },
    storageGiB: { type: 'integer', minimum: 1, maximum: 100 },
    maxLifetimeSeconds: { type: 'integer', minimum: 1, maximum: 86400 },
    defaultCommandTimeoutSeconds: { type: 'integer', minimum: 1, maximum: 900 },
    environment: environmentSchema,
    repository: repositorySchema,
    snapshotId: idSchema,
  },
};
const commandRequestSchema: JsonSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['command'],
  properties: {
    command: { type: 'string', minLength: 1, maxLength: 32768 },
    cwd: workspacePathSchema,
    environment: environmentSchema,
    timeoutSeconds: { type: 'integer', minimum: 1, maximum: 900 },
  },
};
const emptyResultSchema: JsonSchema = { type: 'object', additionalProperties: false };
const sandboxResultSchema: JsonSchema = { type: 'object', additionalProperties: false, required: ['sandbox'], properties: { sandbox: { type: 'object' } } };
const commandResultSchema: JsonSchema = { type: 'object', additionalProperties: false, required: ['sandboxId', 'commandId', 'result'], properties: { sandboxId: idSchema, commandId: idSchema, result: { type: 'object' } } };
const eventResultSchema: JsonSchema = { type: 'object', additionalProperties: false, required: ['sandboxId', 'commandId', 'event'], properties: { sandboxId: idSchema, commandId: idSchema, event: { type: 'object' } } };
const fileResultSchema: JsonSchema = { type: 'object', additionalProperties: false, required: ['sandboxId', 'path', 'content'], properties: { sandboxId: idSchema, path: filePathSchema, content: { type: 'string', maxLength: 13981016 } } };
const filesResultSchema: JsonSchema = { type: 'object', additionalProperties: false, required: ['sandboxId', 'path', 'entries'], properties: { sandboxId: idSchema, path: workspacePathSchema, entries: { type: 'array', items: { type: 'object' } } } };
const snapshotResultSchema: JsonSchema = { type: 'object', additionalProperties: false, required: ['snapshot'], properties: { snapshot: { type: 'object' } } };

export const toolDefinitions: readonly ToolDefinition[] = [
  { name: 'sandbox_create', description: 'Create a temporary Linux computer.', inputSchema: { type: 'object', additionalProperties: false, properties: { config: sandboxConfigSchema } }, resultSchema: sandboxResultSchema },
  { name: 'sandbox_get', description: 'Inspect one sandbox owned by the caller.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId'], properties: { sandboxId: idSchema } }, resultSchema: sandboxResultSchema },
  { name: 'sandbox_list', description: 'List sandboxes owned by the caller.', inputSchema: { type: 'object', additionalProperties: false, properties: { cursor: { type: 'string', minLength: 1, maxLength: 512 }, limit: { type: 'integer', minimum: 1, maximum: 100 }, state: { type: 'string', enum: ['requested', 'provisioning', 'starting', 'running', 'snapshotting', 'stopping', 'stopped', 'failed', 'destroyed'] } } }, resultSchema: { type: 'object', additionalProperties: false, required: ['items'], properties: { items: { type: 'array', items: { type: 'object' } }, nextCursor: { type: 'string', maxLength: 512 } } } },
  { name: 'sandbox_exec', description: 'Submit one bounded command to a running sandbox.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId', 'request'], properties: { sandboxId: idSchema, request: commandRequestSchema } }, resultSchema: commandResultSchema },
  { name: 'sandbox_exec_stream', description: 'Expose ordered command events and terminal status.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId', 'commandId'], properties: { sandboxId: idSchema, commandId: idSchema, lastEventId: { type: 'string', minLength: 1, maxLength: 128, pattern: '^[0-9]+$' } } }, resultSchema: eventResultSchema },
  { name: 'sandbox_read_file', description: 'Read bounded text from a workspace file.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId', 'path'], properties: { sandboxId: idSchema, path: filePathSchema } }, resultSchema: fileResultSchema },
  { name: 'sandbox_write_file', description: 'Write bounded text to a workspace file.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId', 'path', 'content'], properties: { sandboxId: idSchema, path: filePathSchema, content: { type: 'string', maxLength: 13981016 } } }, resultSchema: { type: 'object', additionalProperties: false, required: ['sandboxId', 'entry'], properties: { sandboxId: idSchema, entry: { type: 'object' } } } },
  { name: 'sandbox_list_files', description: 'List workspace entries below a bounded path.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId'], properties: { sandboxId: idSchema, path: workspacePathSchema } }, resultSchema: filesResultSchema },
  { name: 'sandbox_delete_file', description: 'Delete a workspace file or empty directory.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId', 'path'], properties: { sandboxId: idSchema, path: filePathSchema } }, resultSchema: emptyResultSchema },
  { name: 'sandbox_snapshot', description: 'Save the workspace as a snapshot.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId'], properties: { sandboxId: idSchema, expiresInSeconds: { type: 'integer', minimum: 1, maximum: 604800 } } }, resultSchema: snapshotResultSchema },
  { name: 'sandbox_restore', description: 'Restore a snapshot into a fresh sandbox.', inputSchema: { type: 'object', additionalProperties: false, required: ['snapshotId'], properties: { snapshotId: idSchema } }, resultSchema: sandboxResultSchema },
  { name: 'sandbox_destroy', description: 'Request idempotent destruction of a sandbox.', inputSchema: { type: 'object', additionalProperties: false, required: ['sandboxId'], properties: { sandboxId: idSchema } }, resultSchema: sandboxResultSchema },
];

export function getToolDefinition(name: string): ToolDefinition | undefined {
  return toolDefinitions.find((tool) => tool.name === name);
}

export function structuredResult<T>(value: T): ToolResult<T> {
  return { content: [{ type: 'text', text: JSON.stringify(value) }], structuredContent: value };
}

export function createToolRegistry(handlers: Partial<Record<ToolName, ToolHandler>> = {}) {
  return {
    definitions: toolDefinitions,
    async call(name: string, arguments_: unknown, context: ToolContext): Promise<ToolResult> {
      const validated = validateToolArguments(name, arguments_);
      const handler = handlers[name as ToolName];
      if (!handler) throw new McpError('tool_not_implemented', `MCP tool ${name} is not implemented yet.`);
      return handler(validated, context);
    },
  };
}
