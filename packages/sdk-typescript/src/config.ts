export interface SandboxClientConfig {
  baseUrl: string;
  apiKey: string;
  fetch?: typeof fetch;
  requestTimeoutMs?: number;
}

export interface NormalizedConfig {
  baseUrl: string;
  apiKey: string;
  fetch: typeof fetch;
  requestTimeoutMs: number;
}

const defaultRequestTimeoutMs = 30_000;

export function normalizeConfig(config: SandboxClientConfig): NormalizedConfig {
  if (!config || typeof config !== 'object') {
    throw new TypeError('SandboxClient configuration is required.');
  }
  if (typeof config.baseUrl !== 'string' || config.baseUrl.trim() === '') {
    throw new TypeError('SandboxClient baseUrl is required.');
  }
  if (typeof config.apiKey !== 'string' || config.apiKey.trim() === '') {
    throw new TypeError('SandboxClient apiKey is required.');
  }
  const requestTimeoutMs = config.requestTimeoutMs ?? defaultRequestTimeoutMs;
  if (!Number.isFinite(requestTimeoutMs) || requestTimeoutMs <= 0) {
    throw new TypeError('SandboxClient requestTimeoutMs must be greater than zero.');
  }
  const fetchImplementation = config.fetch ?? globalThis.fetch;
  if (typeof fetchImplementation !== 'function') {
    throw new TypeError('SandboxClient requires fetch in this runtime.');
  }
  return {
    baseUrl: config.baseUrl.replace(/\/+$/, ''),
    apiKey: config.apiKey,
    fetch: fetchImplementation,
    requestTimeoutMs,
  };
}
