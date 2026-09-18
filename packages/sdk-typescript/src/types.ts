import type { components } from '@haedes/api-types';

export type Sandbox = components['schemas']['Sandbox'];
export type SandboxState = components['schemas']['SandboxState'];
export type CommandResult = components['schemas']['CommandResult'];
export type FileEntry = components['schemas']['FileEntry'];
export type SnapshotMetadata = components['schemas']['SnapshotMetadata'];
export type RepositoryConfig = components['schemas']['RepositoryConfig'];
export type CommandEvent =
  | { type: 'started'; commandId: string; at: string }
  | { type: 'stdout' | 'stderr'; commandId: string; data: string; at: string }
  | { type: 'completed'; commandId: string; result: CommandResult; at: string }
  | { type: 'failed'; commandId: string; code: string; message: string; at: string };

export interface Page<T> {
  items: T[];
  nextCursor?: string;
}

export interface CreateSandboxInput {
  image?: string;
  cpuMillis?: number;
  memoryMiB?: number;
  storageGiB?: number;
  maxLifetimeSeconds?: number;
  defaultCommandTimeoutSeconds?: number;
  environment?: Record<string, string>;
  repository?: RepositoryConfig;
  snapshotId?: string;
}

export interface ExecOptions {
  cwd?: string;
  environment?: Record<string, string>;
  timeoutSeconds?: number;
  signal?: AbortSignal;
}

export interface SnapshotOptions {
  expiresInSeconds?: number;
}

export interface ListSandboxesOptions {
  cursor?: string;
  limit?: number;
  state?: SandboxState;
}

export type FileContent = string | Uint8Array;
