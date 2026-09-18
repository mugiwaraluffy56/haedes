import { SandboxClient, type SandboxClientConfig } from '@haedes/sdk';

export interface PlatformClient {
  readonly sdk: SandboxClient;
}

export function createPlatformClient(config: SandboxClientConfig): PlatformClient {
  return { sdk: new SandboxClient(config) };
}
