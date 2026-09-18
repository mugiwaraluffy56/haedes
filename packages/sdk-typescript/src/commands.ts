import type { CommandResult, ExecOptions } from './types.js';

export function commandPayload(command: string, options: ExecOptions = {}) {
  if (typeof command !== 'string' || command.length === 0) {
    throw new TypeError('command must not be empty.');
  }
  return {
    command,
    ...(options.cwd === undefined ? {} : { cwd: options.cwd }),
    ...(options.environment === undefined ? {} : { environment: options.environment }),
    ...(options.timeoutSeconds === undefined ? {} : { timeoutSeconds: options.timeoutSeconds }),
  };
}

export type { CommandResult };
