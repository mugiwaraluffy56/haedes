import { McpError } from './errors.js';
import { getToolDefinition, type JsonSchema } from './tools.js';

const forbiddenKeys = new Set([
  'apiKey', 'api_key', 'authorization', 'awsAccessKeyId', 'awsSecretAccessKey',
  'runtimeToken', 'runtime_token', 'taskArn', 'privateIp', 'ecsTaskArn', 'lifecycleState',
]);

function fail(path: string, message: string): never {
  throw new McpError('invalid_tool_arguments', `${path}: ${message}`);
}

function validateValue(value: unknown, schema: JsonSchema, path: string): void {
  if (schema.const !== undefined && value !== schema.const) fail(path, 'must use the documented constant.');
  if (schema.enum && !schema.enum.includes(value)) fail(path, 'contains an unsupported value.');
  if (schema.type === 'object') {
    if (typeof value !== 'object' || value === null || Array.isArray(value)) fail(path, 'must be an object.');
    const record = value as Record<string, unknown>;
    if (schema.maxProperties !== undefined && Object.keys(record).length > schema.maxProperties) fail(path, 'has too many properties.');
    for (const key of Object.keys(record)) {
      if (forbiddenKeys.has(key)) fail(`${path}.${key}`, 'is controlled by the server and cannot be supplied by a tool caller.');
    }
    for (const key of schema.required ?? []) if (!(key in record)) fail(path, `is missing required property ${key}.`);
    for (const [key, child] of Object.entries(record)) {
      const childSchema = schema.properties?.[key] ?? (typeof schema.additionalProperties === 'object' ? schema.additionalProperties : undefined);
      if (!childSchema && schema.additionalProperties === false) fail(`${path}.${key}`, 'is not a documented property.');
      if (childSchema) validateValue(child, childSchema, `${path}.${key}`);
    }
    return;
  }
  if (schema.type === 'array') {
    if (!Array.isArray(value)) fail(path, 'must be an array.');
    if (schema.items) value.forEach((item, index) => validateValue(item, schema.items!, `${path}[${index}]`));
    return;
  }
  if (schema.type === 'string') {
    if (typeof value !== 'string') fail(path, 'must be a string.');
    if (schema.minLength !== undefined && value.length < schema.minLength) fail(path, 'is too short.');
    if (schema.maxLength !== undefined && value.length > schema.maxLength) fail(path, 'is too long.');
    if (schema.pattern && !new RegExp(schema.pattern).test(value)) fail(path, 'has an invalid format.');
    return;
  }
  if (schema.type === 'integer') {
    if (!Number.isInteger(value)) fail(path, 'must be an integer.');
    if (schema.minimum !== undefined && (value as number) < schema.minimum) fail(path, 'is below the minimum.');
    if (schema.maximum !== undefined && (value as number) > schema.maximum) fail(path, 'is above the maximum.');
    return;
  }
  if (schema.type === 'number' && (typeof value !== 'number' || !Number.isFinite(value))) fail(path, 'must be a finite number.');
  if (schema.type === 'boolean' && typeof value !== 'boolean') fail(path, 'must be a boolean.');
  if (schema.type === 'null' && value !== null) fail(path, 'must be null.');
}

export function validateToolArguments(name: string, arguments_: unknown): Record<string, unknown> {
  const definition = getToolDefinition(name);
  if (!definition) throw new McpError('tool_not_found', `Unknown MCP tool ${name}.`);
  validateValue(arguments_, definition.inputSchema, 'arguments');
  return arguments_ as Record<string, unknown>;
}
