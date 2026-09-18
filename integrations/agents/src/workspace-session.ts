import { SandboxClient, type CommandEvent, type CreateSandboxInput, type ExecOptions, type SandboxHandle, type SnapshotMetadata } from '@haedes/sdk';

export interface AgentSandboxSession {
  create(input?: CreateSandboxInput): Promise<SandboxHandle>;
  run(command: string, options?: ExecOptions): AsyncIterable<CommandEvent>;
  read(path: string): Promise<string>;
  write(path: string, content: string): Promise<void>;
  save(): Promise<SnapshotMetadata>;
  restore(snapshotId: string): Promise<SandboxHandle>;
  close(): Promise<void>;
}

/**
 * A host-facing sandbox contract for examples and adapters. It deliberately
 * contains no model, planning, tool-selection, or lifecycle state machine.
 */
export class WorkspaceSession implements AgentSandboxSession {
  private sandbox: SandboxHandle | null = null;

  constructor(private readonly client: SandboxClient) {}

  async create(input: CreateSandboxInput = {}): Promise<SandboxHandle> {
    if (this.sandbox) throw new Error('A sandbox is already attached to this session.');
    this.sandbox = await this.client.sandboxes.create(input);
    return this.sandbox;
  }

  run(command: string, options: ExecOptions = {}): AsyncIterable<CommandEvent> {
    return this.requireSandbox().execStream(command, options);
  }

  async read(path: string): Promise<string> {
    return this.requireSandbox().readFile(path);
  }

  async write(path: string, content: string): Promise<void> {
    await this.requireSandbox().writeFile(path, content);
  }

  async save(): Promise<SnapshotMetadata> {
    return this.requireSandbox().snapshot();
  }

  async restore(snapshotId: string): Promise<SandboxHandle> {
    if (this.sandbox) throw new Error('Close the current sandbox before restoring a new one.');
    this.sandbox = await this.client.sandboxes.create({ snapshotId });
    return this.sandbox;
  }

  async close(): Promise<void> {
    if (!this.sandbox) return;
    const sandbox = this.sandbox;
    await sandbox.destroy();
    this.sandbox = null;
  }

  private requireSandbox(): SandboxHandle {
    if (!this.sandbox) throw new Error('Create or restore a sandbox before using the session.');
    return this.sandbox;
  }
}
