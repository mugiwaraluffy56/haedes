import assert from 'node:assert/strict';
import { WorkspaceSession } from '../src/workspace-session.js';
import type { SandboxClient, SandboxHandle } from '@haedes/sdk';

const calls: string[] = [];

function fakeHandle(id: string): SandboxHandle {
  return {
    id,
    async *execStream(command: string) {
      calls.push(`run:${command}`);
      yield { type: 'started', commandId: `cmd_${id}`, at: '2026-01-01T00:00:00Z' };
      yield { type: 'completed', commandId: `cmd_${id}`, result: { id: `cmd_${id}`, sandboxId: id, command, exitCode: 0, startedAt: '2026-01-01T00:00:00Z', timedOut: false }, at: '2026-01-01T00:00:01Z' };
    },
    async readFile(path: string) { calls.push(`read:${path}`); return 'fixture'; },
    async writeFile(path: string) { calls.push(`write:${path}`); },
    async snapshot() { calls.push('snapshot'); return { id: 'snap_1', sandboxId: id, state: 'ready', createdAt: '2026-01-01T00:00:00Z', expiresAt: '2026-01-02T00:00:00Z' }; },
    async destroy() { calls.push(`destroy:${id}`); },
  } as unknown as SandboxHandle;
}

const handles = [fakeHandle('sbx_1'), fakeHandle('sbx_2')];
const client = {
  sandboxes: {
    async create(input: unknown) {
      calls.push(`create:${JSON.stringify(input)}`);
      return handles.shift();
    },
  },
} as unknown as SandboxClient;

const session = new WorkspaceSession(client);
const created = await session.create({ image: 'fixture' });
assert.equal(created.id, 'sbx_1');
const events = [];
for await (const event of session.run('npm test')) events.push(event);
assert.deepEqual(events.map((event) => event.type), ['started', 'completed']);
await session.write('/workspace/file.txt', 'fixture');
assert.equal(await session.read('/workspace/file.txt'), 'fixture');
assert.equal((await session.save()).id, 'snap_1');
await assert.rejects(session.restore('snap_1'), /Close the current sandbox/);
await session.close();

const restored = await session.restore('snap_1');
assert.equal(restored.id, 'sbx_2');
await session.close();
assert.deepEqual(calls, [
  'create:{"image":"fixture"}',
  'run:npm test',
  'write:/workspace/file.txt',
  'read:/workspace/file.txt',
  'snapshot',
  'destroy:sbx_1',
  'create:{"snapshotId":"snap_1"}',
  'destroy:sbx_2',
]);
console.log('agent session contract checks passed');
