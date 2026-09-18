import type { CommandResult, ExecOptions } from './types.js';

const maxCommandLength = 16 * 1024;
const maxEnvironmentEntries = 32;
const maxEnvironmentValueLength = 4096;
const maxTimeoutSeconds = 900;

export interface CommandPayload {
  command: string;
  cwd?: string;
  environment?: Record<string, string>;
  timeoutSeconds?: number;
}

export function commandPayload(command: string, options: ExecOptions = {}): CommandPayload {
  if (typeof command !== 'string' || command.length === 0) {
    throw new TypeError('command must not be empty.');
  }

  if (command.length > maxCommandLength) {
    throw new TypeError(`command must not exceed ${maxCommandLength} characters.`);
  }

  if (options.cwd !== undefined && (typeof options.cwd !== 'string' || options.cwd.length === 0)) {
    throw new TypeError('cwd must be a non-empty string when provided.');
  }

  if (options.environment !== undefined) {
    const entries = Object.entries(options.environment);
    if (entries.length > maxEnvironmentEntries) {
      throw new TypeError(`environment must not contain more than ${maxEnvironmentEntries} entries.`);
    }
    for (const [key, value] of entries) {
      if (key.length === 0 || typeof value !== 'string') {
        throw new TypeError('environment keys must be non-empty and values must be strings.');
      }
      if (value.length > maxEnvironmentValueLength) {
        throw new TypeError(`environment values must not exceed ${maxEnvironmentValueLength} characters.`);
      }
    }
  }

  if (
    options.timeoutSeconds !== undefined &&
    (!Number.isInteger(options.timeoutSeconds) || options.timeoutSeconds < 1 || options.timeoutSeconds > maxTimeoutSeconds)
  ) {
    throw new TypeError(`timeoutSeconds must be an integer between 1 and ${maxTimeoutSeconds}.`);
  }

  return {
    command,
    ...(options.cwd === undefined ? {} : { cwd: options.cwd }),
    ...(options.environment === undefined ? {} : { environment: options.environment }),
    ...(options.timeoutSeconds === undefined ? {} : { timeoutSeconds: options.timeoutSeconds }),
  };
}

export type { CommandResult };
