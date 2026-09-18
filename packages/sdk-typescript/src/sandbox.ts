import { fileBody } from './filesystem.js';
import { snapshotPayload } from './snapshots.js';
import { SandboxClient } from './client.js';
import type {
  CommandEvent,
  CommandResult,
  CreateSandboxInput,
  ExecOptions,
  FileContent,
  FileEntry,
  ListSandboxesOptions,
  Page,
  Sandbox,
  SnapshotMetadata,
  SnapshotOptions,
} from './types.js';

const defaultSandboxConfig = {
  image: 'haedes-sandbox-dev:dev',
  cpuMillis: 512,
  memoryMiB: 1024,
  storageGiB: 10,
  maxLifetimeSeconds: 600,
  defaultCommandTimeoutSeconds: 60,
  environment: {},
};

export class SandboxCollection {
  constructor(private readonly client: SandboxClient) {}

  async create(config: CreateSandboxInput = {}): Promise<SandboxHandle> {
    const sandbox = await this.client.requestJson<Sandbox>('/v1/sandboxes', {
      method: 'POST',
      headers: { 'Idempotency-Key': SandboxClient.idempotencyKey() },
      body: JSON.stringify({ config: { ...defaultSandboxConfig, ...config } }),
    });
    return new SandboxHandle(this.client, sandbox.id);
  }

  async get(id: string): Promise<SandboxHandle> {
    const sandbox = await this.client.requestJson<Sandbox>(`/v1/sandboxes/${encodeURIComponent(id)}`);
    return new SandboxHandle(this.client, sandbox.id);
  }

  async list(options: ListSandboxesOptions = {}): Promise<Page<Sandbox>> {
    const query = new URLSearchParams();
    if (options.cursor !== undefined) query.set('cursor', options.cursor);
    if (options.limit !== undefined) query.set('limit', String(options.limit));
    if (options.state !== undefined) query.set('state', options.state);
    const suffix = query.size === 0 ? '' : `?${query.toString()}`;
    const page = await this.client.requestJson<{ items: Sandbox[]; page: { nextCursor?: string; hasMore: boolean } }>(`/v1/sandboxes${suffix}`);
    return { items: page.items, ...(page.page.nextCursor ? { nextCursor: page.page.nextCursor } : {}) };
  }
}

export class SandboxHandle {
  constructor(private readonly client: SandboxClient, readonly id: string) {}

  async get(): Promise<Sandbox> {
    return this.client.requestJson<Sandbox>(`/v1/sandboxes/${encodeURIComponent(this.id)}`);
  }

  async exec(command: string, options: ExecOptions = {}): Promise<CommandResult> {
    return this.client.createCommand(this.id, command, options);
  }

  execStream(command: string, options: ExecOptions = {}): AsyncGenerator<CommandEvent> {
    const commandPromise = this.exec(command, options);
    return this.streamAfterCommand(commandPromise, options.signal);
  }

  private async *streamAfterCommand(commandPromise: Promise<CommandResult>, signal?: AbortSignal): AsyncGenerator<CommandEvent> {
    const command = await commandPromise;
    yield* this.client.streamCommandEvents(this.id, command.id, { signal });
  }

  async readFileBytes(path: string): Promise<Uint8Array> {
    return this.client.requestBytes(this.filePath('content', path));
  }

  async readFile(path: string): Promise<string> {
    return new TextDecoder().decode(await this.readFileBytes(path));
  }

  async writeFile(path: string, content: FileContent): Promise<void> {
    await this.client.requestEmpty(this.filePath('content', path), { method: 'PUT', body: fileBody(content) as unknown as BodyInit, headers: { 'Content-Type': 'application/octet-stream' } });
  }

  async listFiles(path = '/workspace'): Promise<FileEntry[]> {
    const query = new URLSearchParams({ path });
    const result = await this.client.requestJson<{ entries: FileEntry[] }>(`/v1/sandboxes/${encodeURIComponent(this.id)}/files?${query}`);
    return result.entries;
  }

  async deleteFile(path: string): Promise<void> {
    await this.client.requestEmpty(this.filePath('content', path), { method: 'DELETE' });
  }

  async snapshot(options: SnapshotOptions = {}): Promise<SnapshotMetadata> {
    return this.client.requestJson<SnapshotMetadata>(`/v1/sandboxes/${encodeURIComponent(this.id)}/snapshots`, {
      method: 'POST',
      headers: { 'Idempotency-Key': SandboxClient.idempotencyKey() },
      body: JSON.stringify(snapshotPayload(options)),
    });
  }

  async restore(snapshotId: string): Promise<void> {
    if (typeof snapshotId !== 'string' || snapshotId.length === 0) {
      throw new TypeError('snapshotId must not be empty.');
    }
    await this.client.requestEmpty(`/v1/sandboxes/${encodeURIComponent(this.id)}/restore`, {
      method: 'POST',
      body: JSON.stringify({ snapshotId }),
    });
  }

  async destroy(): Promise<Sandbox> {
    return this.client.requestJson<Sandbox>(`/v1/sandboxes/${encodeURIComponent(this.id)}`, { method: 'DELETE' });
  }

  private filePath(resource: string, path: string): string {
    return `/v1/sandboxes/${encodeURIComponent(this.id)}/files/${resource}?path=${encodeURIComponent(path)}`;
  }
}
