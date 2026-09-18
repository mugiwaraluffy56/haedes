import { createAgentSandboxSession } from '@haedes/agent-adapter';
import { SandboxClient, type CommandEvent } from '@haedes/sdk';

function required(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`Set ${name} before running the coding-agent example.`);
  return value;
}

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", "'\\''")}'`;
}

async function runTests(session: ReturnType<typeof createAgentSandboxSession>, repositoryPath: string): Promise<CommandEvent[]> {
  const events: CommandEvent[] = [];
  for await (const event of session.run(`cd ${shellQuote(repositoryPath)} && npm test`)) events.push(event);
  return events;
}

const repositoryPath = process.env.HAEDES_REPOSITORY_PATH ?? '/workspace/fixture';
const client = new SandboxClient({ baseUrl: required('HAEDES_API_URL'), apiKey: required('HAEDES_API_KEY') });
const session = createAgentSandboxSession(client);
const sandbox = await session.create({
  repository: {
    provider: 'github',
    url: required('HAEDES_REPOSITORY_URL'),
    path: repositoryPath,
    ...(process.env.HAEDES_REPOSITORY_REF ? { ref: process.env.HAEDES_REPOSITORY_REF } : {}),
  },
});

let snapshotId: string | undefined;
let restoredSandboxId: string | undefined;
try {
  const failingEvents = await runTests(session, repositoryPath);
  const sourcePath = `${repositoryPath}/src/auth.ts`;
  const source = await session.read(sourcePath);
  const fixedSource = source.replace('return username === password;', 'return users[username] === password;');
  if (fixedSource === source) throw new Error('The fixture did not contain its expected deterministic bug.');
  await session.write(sourcePath, fixedSource);
  const passingEvents = await runTests(session, repositoryPath);
  const snapshot = await session.save();
  snapshotId = snapshot.id;
  await session.close();

  const restored = await session.restore(snapshot.id);
  restoredSandboxId = restored.id;
  const restoredEvents = await runTests(session, repositoryPath);
  const terminal = (events: CommandEvent[]) => events.at(-1)?.type;
  console.log(JSON.stringify({
    originalSandboxId: sandbox.id,
    firstTestStatus: terminal(failingEvents),
    passingTestStatus: terminal(passingEvents),
    snapshotId,
    restoredSandboxId,
    restoredTestStatus: terminal(restoredEvents),
  }, null, 2));
} finally {
  await session.close();
}
