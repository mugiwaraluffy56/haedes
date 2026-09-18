#!/usr/bin/env node

import assert from 'node:assert/strict';
import { performance } from 'node:perf_hooks';

const apiUrl = (process.env.HAEDES_API_URL ?? 'http://127.0.0.1:18080').replace(/\/$/, '');
const apiKey = process.env.HAEDES_API_KEY ?? 'local-api-key';
const virtualUsers = boundedInteger('HAEDES_LOAD_VIRTUAL_USERS', 4, 1, 32);
const commandsPerSandbox = boundedInteger('HAEDES_LOAD_COMMANDS', 2, 1, 16);
const lifecycleRounds = boundedInteger('HAEDES_LOAD_LIFECYCLE_ROUNDS', 2, 1, 16);
const dryRun = process.argv.includes('--dry-run');
const activeSandboxes = new Set();
const measurements = { createMs: [], commandMs: [], eventMs: [], destroyMs: [], failures: 0 };

function boundedInteger(name, fallback, minimum, maximum) {
  const value = Number.parseInt(process.env[name] ?? String(fallback), 10);
  if (!Number.isInteger(value) || value < minimum || value > maximum) {
    throw new Error(`${name} must be an integer between ${minimum} and ${maximum}`);
  }
  return value;
}

function plan() {
  return { environment: process.env.HAEDES_ENV ?? 'local', apiUrl, virtualUsers, commandsPerSandbox, lifecycleRounds };
}

async function request(path, options = {}) {
  const response = await fetch(`${apiUrl}${path}`, {
    ...options,
    headers: { Authorization: `Bearer ${apiKey}`, ...(options.headers ?? {}) },
  });
  const body = response.status === 204 ? null : await response.text();
  if (!response.ok) throw new Error(`${options.method ?? 'GET'} ${path} returned ${response.status}: ${body}`);
  return body ? JSON.parse(body) : null;
}

async function createSandbox(round, user) {
  const started = performance.now();
  const value = await request('/v1/sandboxes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `load-${round}-${user}-${Date.now()}` },
    body: JSON.stringify({ config: { image: 'haedes-sandbox-dev:dev', cpuMillis: 512, memoryMiB: 1024, storageGiB: 10, maxLifetimeSeconds: 600, defaultCommandTimeoutSeconds: 30, environment: { LOAD: 'true' } } }),
  });
  assert.equal(value.state, 'running');
  activeSandboxes.add(value.id);
  measurements.createMs.push(performance.now() - started);
  return value.id;
}

async function runCommand(sandboxId, index) {
  const started = performance.now();
  const command = await request(`/v1/sandboxes/${encodeURIComponent(sandboxId)}/commands`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ command: `printf load-${index}` }),
  });
  measurements.commandMs.push(performance.now() - started);
  const eventStarted = performance.now();
  const response = await fetch(`${apiUrl}/v1/sandboxes/${encodeURIComponent(sandboxId)}/commands/${encodeURIComponent(command.id)}/events`, {
    headers: { Authorization: `Bearer ${apiKey}` },
  });
  const body = await response.text();
  if (!response.ok || !body.includes('event: completed')) throw new Error(`command ${command.id} did not complete: ${body}`);
  measurements.eventMs.push(performance.now() - eventStarted);
}

async function destroySandbox(sandboxId) {
  const started = performance.now();
  const value = await request(`/v1/sandboxes/${encodeURIComponent(sandboxId)}`, { method: 'DELETE' });
  assert.ok(['stopping', 'stopped', 'destroyed'].includes(value.state));
  measurements.destroyMs.push(performance.now() - started);
  activeSandboxes.delete(sandboxId);
}

async function runUser(user) {
  for (let round = 0; round < lifecycleRounds; round += 1) {
    const sandboxId = await createSandbox(round, user);
    await Promise.all(Array.from({ length: commandsPerSandbox }, (_, index) => runCommand(sandboxId, index)));
    await destroySandbox(sandboxId);
  }
}

function summary(values) {
  if (values.length === 0) return { count: 0, averageMs: 0, p95Ms: 0 };
  const sorted = [...values].sort((left, right) => left - right);
  return { count: sorted.length, averageMs: Number((sorted.reduce((sum, value) => sum + value, 0) / sorted.length).toFixed(2)), p95Ms: Number(sorted[Math.min(sorted.length - 1, Math.ceil(sorted.length * 0.95) - 1)].toFixed(2)) };
}

if (dryRun) {
  console.log(JSON.stringify({ ...plan(), dryRun: true }, null, 2));
} else {
  try {
    await Promise.all(Array.from({ length: virtualUsers }, (_, user) => runUser(user)));
    const metricsResponse = await fetch(`${apiUrl}/metrics`, { headers: { Authorization: `Bearer ${apiKey}` } });
    console.log(JSON.stringify({ ...plan(), dryRun: false, measurements: { create: summary(measurements.createMs), command: summary(measurements.commandMs), event: summary(measurements.eventMs), destroy: summary(measurements.destroyMs), failures: measurements.failures }, metricsAvailable: metricsResponse.ok }, null, 2));
  } catch (error) {
    measurements.failures += 1;
    console.error(JSON.stringify({ ...plan(), error: error instanceof Error ? error.message : String(error), failures: measurements.failures }, null, 2));
    process.exitCode = 1;
  } finally {
    await Promise.all([...activeSandboxes].map((sandboxId) => request(`/v1/sandboxes/${encodeURIComponent(sandboxId)}`, { method: 'DELETE' }).catch(() => undefined)));
  }
}
