#!/usr/bin/env node

import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { spawn, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const clientName = process.env.HAEDES_AGENT_CLIENT ?? 'claude';
const apiUrl = process.env.HAEDES_API_URL;
const apiKey = process.env.HAEDES_API_KEY;
const repositoryUrl = process.env.HAEDES_REPOSITORY_URL;
const repositoryPath = process.env.HAEDES_REPOSITORY_PATH ?? '/workspace/fixture';
const repositoryRef = process.env.HAEDES_REPOSITORY_REF;
const mcpServer = resolve(root, 'integrations/mcp/dist/server.js');
const dryRun = process.argv.includes('--dry-run');

if (clientName !== 'claude') {
  throw new Error('The compatibility harness currently supports Claude Code; add and verify another client before selecting it.');
}

function required(name, value) {
  if (!value && !dryRun) throw new Error(`Set ${name} before running the coding-agent journey.`);
  return value ?? `<${name}>`;
}

function shellQuote(value) {
  return `'${value.replaceAll("'", "'\\''")}'`;
}

function prompt() {
  return [
    'Use only the haedes MCP tools for this task. Do not edit local files.',
    `Create a sandbox for ${required('HAEDES_REPOSITORY_URL', repositoryUrl)} at ${repositoryPath}${repositoryRef ? ` using ref ${repositoryRef}` : ''}.`,
    `Run npm install in ${shellQuote(repositoryPath)}. Run npm test and record the expected failing result.`,
    `Read ${shellQuote(`${repositoryPath}/src/auth.ts`)}, change only the intentional authentication bug, and write the corrected file.`,
    `Run npm test again and require the tests to pass. Create a snapshot, destroy the original sandbox, restore the snapshot into a new sandbox, and run npm test again.`,
    'Destroy the restored sandbox before finishing.',
    'Return a final JSON object with journeyStatus, originalSandboxId, restoredSandboxId, snapshotId, failedTestObserved, passingTestObserved, restoredTestPassed, and bothSandboxesDestroyed.',
  ].join(' ');
}

function claudeConfig() {
  if (!existsSync(mcpServer) && !dryRun) throw new Error('Build @haedes/mcp before running the journey.');
  const directory = mkdtempSync(resolve(tmpdir(), 'haedes-agent-'));
  const path = resolve(directory, 'mcp.json');
  const config = {
    mcpServers: {
      haedes: {
        command: process.execPath,
        args: [mcpServer],
        env: { HAEDES_API_URL: required('HAEDES_API_URL', apiUrl), HAEDES_API_KEY: required('HAEDES_API_KEY', apiKey) },
      },
    },
  };
  writeFileSync(path, JSON.stringify(config, null, 2));
  return { directory, path };
}

const config = claudeConfig();
const clientVersion = spawnSync(clientName, ['--version'], { encoding: 'utf8' });
if (clientVersion.error) throw clientVersion.error;
const invocation = [
  clientName,
  '--print',
  '--output-format', 'json',
  '--strict-mcp-config',
  '--mcp-config', config.path,
  '--permission-mode', 'dontAsk',
  '--tools', '',
  '--no-session-persistence',
  prompt(),
];

console.error(JSON.stringify({
  client: clientName,
  version: clientVersion.stdout.trim(),
  mcpConfig: readFileSync(config.path, 'utf8'),
  invocation,
  dryRun,
}, null, 2));

if (dryRun) {
  rmSync(config.directory, { recursive: true, force: true });
  process.exit(0);
}

const child = spawn(invocation[0], invocation.slice(1), { cwd: root, stdio: ['ignore', 'pipe', 'inherit'] });
let output = '';
child.stdout.on('data', (chunk) => {
  output += chunk;
  process.stdout.write(chunk);
});
const exitCode = await new Promise((resolveExit) => child.once('close', resolveExit));
rmSync(config.directory, { recursive: true, force: true });

if (exitCode !== 0) process.exit(exitCode ?? 1);
if (!/journeyStatus["']?\s*:\s*["']passed/i.test(output)) {
  throw new Error('The coding agent did not report journeyStatus=passed; inspect its output before claiming compatibility.');
}
