import type { components } from './generated';

export * from './generated';
export type { components } from './generated';

export type ApiError = components['schemas']['ApiError'];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function parseApiError(value: unknown): ApiError | undefined {
  if (!isRecord(value) || !isRecord(value.error)) {
    return undefined;
  }

  const error = value.error;
  if (
    typeof error.code !== 'string' ||
    typeof error.message !== 'string' ||
    typeof error.requestId !== 'string' ||
    !isRecord(error.details)
  ) {
    return undefined;
  }

  return {
    code: error.code,
    message: error.message,
    requestId: error.requestId,
    details: error.details,
  };
}
