'use client';

import { SandboxClient, SandboxError, type Page, type Sandbox, type SandboxHandle, type SandboxState, type SnapshotMetadata } from '@haedes/sdk';

export const nonTerminalStates: readonly SandboxState[] = [
  'requested',
  'provisioning',
  'starting',
  'running',
  'snapshotting',
  'stopping',
];

export class DashboardConfigurationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'DashboardConfigurationError';
  }
}

export interface DashboardError {
  message: string;
  requestId?: string;
  code?: string;
  status?: number;
}

export function isNonTerminal(state: SandboxState): boolean {
  return nonTerminalStates.includes(state);
}

export function toDashboardError(error: unknown): DashboardError {
  if (error instanceof SandboxError) {
    return {
      message: error.message,
      requestId: error.requestId,
      code: error.code,
      status: error.status,
    };
  }
  if (error instanceof DashboardConfigurationError || error instanceof Error) {
    return { message: error.message };
  }
  return { message: 'The dashboard could not reach the sandbox API.' };
}

function createClient(): SandboxClient {
  const baseUrl = process.env.NEXT_PUBLIC_HAEDES_API_URL;
  const apiKey = process.env.NEXT_PUBLIC_HAEDES_API_KEY;
  if (!baseUrl || !apiKey) {
    throw new DashboardConfigurationError(
      'Configure NEXT_PUBLIC_HAEDES_API_URL and NEXT_PUBLIC_HAEDES_API_KEY before using the dashboard.',
    );
  }
  return new SandboxClient({ baseUrl, apiKey });
}

export async function listSandboxes(): Promise<Page<Sandbox>> {
  return createClient().sandboxes.list({ limit: 100 });
}

export async function getSandbox(id: string): Promise<Sandbox> {
  const handle = await createClient().sandboxes.get(id);
  return handle.get();
}

export async function getSandboxHandle(id: string): Promise<SandboxHandle> {
  return createClient().sandboxes.get(id);
}

export async function createSnapshot(id: string): Promise<SnapshotMetadata> {
  return (await getSandboxHandle(id)).snapshot();
}

export async function restoreSnapshot(snapshotId: string): Promise<Sandbox> {
  const handle = await createClient().sandboxes.create({ snapshotId });
  return handle.get();
}

export async function destroySandbox(id: string): Promise<void> {
  await (await getSandboxHandle(id)).destroy();
}
