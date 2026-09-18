import { createAgentSandboxSession } from '@haedes/agent-adapter';
import { SandboxClient, type CommandEvent } from '@haedes/sdk';

function required(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`Set ${name} before running the basic sandbox example.`);
  return value;
}

async function collect(events: AsyncIterable<CommandEvent>): Promise<CommandEvent[]> {
  const collected: CommandEvent[] = [];
  for await (const event of events) collected.push(event);
  return collected;
}

const client = new SandboxClient({ baseUrl: required('HAEDES_API_URL'), apiKey: required('HAEDES_API_KEY') });
const session = createAgentSandboxSession(client);
const sandbox = await session.create();

try {
  const commandEvents = await collect(session.run('printf "hello from the basic agent\\n"'));
  await session.write('/workspace/basic-agent.txt', 'written through the provider-neutral session');
  const fileContents = await session.read('/workspace/basic-agent.txt');
  const snapshot = await session.save();
  const terminal = commandEvents.at(-1);

  console.log(JSON.stringify({
    sandboxId: sandbox.id,
    commandId: terminal && 'commandId' in terminal ? terminal.commandId : undefined,
    commandStatus: terminal?.type,
    fileContents,
    snapshotId: snapshot.id,
  }, null, 2));
} finally {
  await session.close();
}
