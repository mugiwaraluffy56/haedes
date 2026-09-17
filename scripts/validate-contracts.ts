import { readFileSync, readdirSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
import { parse as parseYaml } from 'yaml';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const failures: string[] = [];
const PUBLIC_SCHEMA_ID = 'https://haedes.dev/schemas/public-api.json';

const readJson = (relativePath: string) => JSON.parse(readFileSync(join(root, relativePath), 'utf8'));
const readYaml = (relativePath: string) => parseYaml(readFileSync(join(root, relativePath), 'utf8'));
const assert = (condition: unknown, message: string) => {
  if (!condition) failures.push(message);
};

const ajv = new Ajv2020({ allErrors: true, strict: false });
addFormats(ajv);

const publicApi = readYaml('internal/contracts/api.openapi.yaml');
publicApi.$id = PUBLIC_SCHEMA_ID;
ajv.addSchema(publicApi, PUBLIC_SCHEMA_ID);

const validateDocument = (schema: object, value: unknown, name: string) => {
  try {
    const validate = ajv.compile(schema);
    if (!validate(value)) failures.push(`${name}: ${ajv.errorsText(validate.errors)}`);
  } catch (error) {
    failures.push(`${name}: schema compilation failed: ${error instanceof Error ? error.message : String(error)}`);
  }
};

const publicSchemas = publicApi.components?.schemas ?? {};
assert(publicApi.openapi === '3.1.0', 'public contract must use OpenAPI 3.1');
assert(publicApi.paths?.['/v1/sandboxes']?.post, 'public contract must define sandbox creation');
assert(publicApi.paths?.['/v1/sandboxes/{sandboxId}/commands']?.post, 'public contract must define command execution');

for (const schemaName of ['SandboxState', 'SandboxID', 'CommandID', 'SnapshotID', 'RequestID', 'Timestamp', 'CommandRequest', 'ApiError']) {
  assert(publicSchemas[schemaName], `public contract is missing ${schemaName}`);
}

const patterns = {
  SandboxID: '^sbx_[A-Za-z0-9_-]+$',
  CommandID: '^cmd_[A-Za-z0-9_-]+$',
  SnapshotID: '^snp_[A-Za-z0-9_-]+$',
  RequestID: '^req_[A-Za-z0-9_-]+$',
};
for (const [schemaName, pattern] of Object.entries(patterns)) {
  assert(publicSchemas[schemaName]?.pattern === pattern, `${schemaName} must keep its opaque ID pattern`);
}
assert(publicSchemas.Timestamp?.format === 'date-time', 'public timestamps must use date-time format');
assert(publicSchemas.ApiError?.required?.includes('code'), 'ApiError must require code');
assert(publicSchemas.ApiError?.required?.includes('message'), 'ApiError must require message');
assert(publicSchemas.ApiError?.required?.includes('requestId'), 'ApiError must require requestId');
assert(publicSchemas.ApiError?.required?.includes('details'), 'ApiError must require details');

const commandRequest = publicSchemas.CommandRequest;
assert(commandRequest?.properties?.command?.maxLength === 16384, 'public command limit must be 16384 bytes');
assert(commandRequest?.properties?.environment?.maxProperties === 32, 'public environment limit must be 32 entries');
assert(commandRequest?.properties?.timeoutSeconds?.maximum === 900, 'public timeout limit must be 900 seconds');
const fileWriteSchema = publicApi.paths?.['/v1/sandboxes/{sandboxId}/files/content']?.put?.requestBody?.content?.['application/octet-stream']?.schema;
assert(fileWriteSchema?.maxLength === 10485760, 'public file body limit must be 10 MiB');

const lifecycleText = readFileSync(join(root, 'internal/contracts/lifecycle.md'), 'utf8');
const expectedStates = ['requested', 'provisioning', 'starting', 'running', 'snapshotting', 'stopping', 'stopped', 'failed', 'destroyed'];
const stateSection = lifecycleText.split('## Legal transitions')[0];
const documentedStates = [...stateSection.matchAll(/^\| ([a-z]+) \|/gm)].map((match) => match[1]);
assert(JSON.stringify(documentedStates) === JSON.stringify(expectedStates), 'lifecycle states must match the canonical state list');

const expectedTransitions = [
  'requested->provisioning', 'requested->failed',
  'provisioning->starting', 'provisioning->failed',
  'starting->running', 'starting->failed',
  'running->snapshotting', 'running->stopping', 'running->failed',
  'snapshotting->running', 'snapshotting->failed',
  'stopping->stopped', 'stopping->destroyed', 'stopping->failed',
  'stopped->destroyed', 'failed->destroyed',
];
const transitionSection = lifecycleText.split('## Legal transitions')[1]?.split('## Retry and idempotency rules')[0] ?? '';
const documentedTransitions = [...transitionSection.matchAll(/^\| ([a-z]+) \| ([a-z]+) \|/gm)].map((match) => `${match[1]}->${match[2]}`);
assert(JSON.stringify(documentedTransitions) === JSON.stringify(expectedTransitions), 'lifecycle transitions must match the canonical transition list');
const legalTransitions = new Set(documentedTransitions);
for (const transition of expectedTransitions) assert(legalTransitions.has(transition), `legal transition missing: ${transition}`);
for (const transition of ['destroyed->running', 'stopped->running', 'running->destroyed', 'snapshotting->stopped']) {
  assert(!legalTransitions.has(transition), `illegal transition documented as legal: ${transition}`);
}

const runtimeSchema = readJson('packages/protocol/src/runtime-v1.schema.json');
assert(runtimeSchema.$id === 'https://haedes.dev/schemas/runtime.v1.schema.json', 'runtime schema must have a stable versioned ID');
assert(runtimeSchema.$defs?.Version?.const === 'runtime.v1', 'runtime messages must be versioned as runtime.v1');
assert(runtimeSchema.$defs?.Timestamp?.format === 'date-time', 'runtime timestamps must use date-time format');
assert(runtimeSchema.$defs?.RequestID?.pattern === patterns.RequestID, 'runtime request IDs must remain opaque');
assert(runtimeSchema.$defs?.CommandID?.pattern === patterns.CommandID, 'runtime command IDs must remain opaque');
assert(runtimeSchema.$defs?.SnapshotID?.pattern === patterns.SnapshotID, 'runtime snapshot IDs must remain opaque');
assert(runtimeSchema.$defs?.CommandFields?.properties?.command?.maxLength === 16384, 'runtime command limit must be 16384 bytes');
assert(runtimeSchema.$defs?.Environment?.maxProperties === 32, 'runtime environment limit must be 32 entries');
assert(runtimeSchema.$defs?.CommandFields?.properties?.timeoutSeconds?.maximum === 900, 'runtime timeout limit must be 900 seconds');
assert(runtimeSchema.$defs?.FileWriteRequest?.properties?.content?.maxLength === 13981016, 'runtime file body limit must be 10 MiB when base64 encoded');

const runtimeValidate = ajv.compile(runtimeSchema);
for (const file of readdirSync(join(root, 'packages/protocol/src/fixtures')).filter((name) => name.endsWith('.json'))) {
  const fixture = readJson(`packages/protocol/src/fixtures/${file}`);
  if (!runtimeValidate(fixture)) failures.push(`runtime fixture ${file}: ${ajv.errorsText(runtimeValidate.errors)}`);
}

for (const [schemaName, fixtureName] of [['sandbox', 'sandbox'], ['command', 'command'], ['snapshot', 'snapshot']]) {
  const schema = readJson(`internal/schemas/${schemaName}.schema.json`);
  validateDocument(schema, readJson(`internal/schemas/fixtures/${fixtureName}.json`), `persisted ${schemaName}`);
}

const mcpContract = readJson('integrations/mcp/src/tool-contract.schema.json');
const expectedTools = ['create_sandbox', 'execute_command', 'stream_command', 'list_files', 'read_file', 'write_file', 'delete_file', 'create_snapshot', 'restore_snapshot', 'destroy_sandbox'];
assert(JSON.stringify(mcpContract.tools?.map((tool: { name: string }) => tool.name)) === JSON.stringify(expectedTools), 'MCP tool names must match the public platform surface');
const publicReferencePrefix = `${PUBLIC_SCHEMA_ID}#/components/`;
const references: string[] = [];
const collectReferences = (value: unknown) => {
  if (Array.isArray(value)) {
    value.forEach(collectReferences);
  } else if (value && typeof value === 'object') {
    for (const [key, child] of Object.entries(value)) {
      if (key === '$ref' && typeof child === 'string') references.push(child);
      else collectReferences(child);
    }
  }
};
collectReferences(mcpContract.tools);
for (const reference of references) assert(reference.startsWith(publicReferencePrefix), `MCP schema must reference canonical public types: ${reference}`);
for (const tool of mcpContract.tools ?? []) {
  try {
    ajv.compile(tool.inputSchema);
    ajv.compile(tool.resultSchema);
  } catch (error) {
    failures.push(`MCP ${tool.name}: schema compilation failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

if (failures.length > 0) {
  console.error(failures.map((failure) => `- ${failure}`).join('\n'));
  process.exit(1);
}

console.log(`validated public, runtime, persisted, and MCP contracts (${expectedTools.length} MCP tools)`);
