import type { SnapshotOptions } from './types.js';

export function snapshotPayload(options: SnapshotOptions = {}): Record<string, string> {
  if (options.expiresInSeconds === undefined) {
    return {};
  }
  if (!Number.isInteger(options.expiresInSeconds) || options.expiresInSeconds <= 0) {
    throw new TypeError('expiresInSeconds must be a positive integer.');
  }
  return { expiresAt: new Date(Date.now() + options.expiresInSeconds * 1000).toISOString() };
}
