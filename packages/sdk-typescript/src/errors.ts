import { parseApiError } from '@haedes/api-types';

export class SandboxError extends Error {
  readonly code: string;
  readonly requestId?: string;
  readonly status: number;
  readonly details: Record<string, unknown>;

  constructor(options: {
    code: string;
    message: string;
    status: number;
    requestId?: string;
    details?: Record<string, unknown>;
  }) {
    super(options.message);
    this.name = 'SandboxError';
    this.code = options.code;
    this.status = options.status;
    this.requestId = options.requestId;
    this.details = options.details ?? {};
  }

  static async fromResponse(response: Response): Promise<SandboxError> {
    let body: unknown;
    try {
      body = await response.json();
    } catch {
      body = undefined;
    }
    const apiError = parseApiError(body);
    return new SandboxError({
      code: apiError?.code ?? 'http_error',
      message: apiError?.message ?? `Sandbox API request failed with status ${response.status}.`,
      status: response.status,
      requestId: response.headers.get('X-Request-ID') ?? apiError?.requestId,
      details: apiError?.details,
    });
  }
}
