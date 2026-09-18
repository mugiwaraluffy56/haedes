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

export interface DashboardClientConfig {
  baseUrl: string;
  apiKey: string;
  fetch?: typeof fetch;
}

export interface DashboardApi {
  listSandboxes(): Promise<Page<Sandbox>>;
  getSandbox(id: string): Promise<Sandbox>;
  getSandboxHandle(id: string): Promise<SandboxHandle>;
  createSnapshot(id: string): Promise<SnapshotMetadata>;
  restoreSnapshot(snapshotId: string): Promise<Sandbox>;
  destroySandbox(id: string): Promise<Sandbox>;
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

export function createDashboardApi(config: DashboardClientConfig): DashboardApi {
  const client = new SandboxClient(config);
  return {
    listSandboxes: () => client.sandboxes.list({ limit: 100 }),
    getSandbox: async (id) => (await client.sandboxes.get(id)).get(),
    getSandboxHandle: (id) => client.sandboxes.get(id),
    createSnapshot: async (id) => (await client.sandboxes.get(id)).snapshot(),
    restoreSnapshot: async (snapshotId) => (await client.sandboxes.create({ snapshotId })).get(),
    destroySandbox: async (id) => (await client.sandboxes.get(id)).destroy(),
  };
}

function configuredDashboardApi(): DashboardApi {
  const baseUrl = process.env.NEXT_PUBLIC_HAEDES_API_URL;
  const apiKey = process.env.NEXT_PUBLIC_HAEDES_API_KEY;
  if (!baseUrl || !apiKey) {
    throw new DashboardConfigurationError(
      'Configure NEXT_PUBLIC_HAEDES_API_URL and NEXT_PUBLIC_HAEDES_API_KEY before using the dashboard.',
    );
  }
  return createDashboardApi({ baseUrl, apiKey });
}

export async function listSandboxes(): Promise<Page<Sandbox>> {
  return configuredDashboardApi().listSandboxes();
}

export async function getSandbox(id: string): Promise<Sandbox> {
  return configuredDashboardApi().getSandbox(id);
}

export async function getSandboxHandle(id: string): Promise<SandboxHandle> {
  return configuredDashboardApi().getSandboxHandle(id);
}

export async function createSnapshot(id: string): Promise<SnapshotMetadata> {
  return configuredDashboardApi().createSnapshot(id);
}

export async function restoreSnapshot(snapshotId: string): Promise<Sandbox> {
  return configuredDashboardApi().restoreSnapshot(snapshotId);
}

export async function destroySandbox(id: string): Promise<Sandbox> {
  return configuredDashboardApi().destroySandbox(id);
}
