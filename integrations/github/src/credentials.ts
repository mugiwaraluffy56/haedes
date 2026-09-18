import type { GitHubRepository } from './repository-url.js';
import { GitHubRepositoryError } from './repository-url.js';

export interface GitHubCredentialProvider {
  getToken(repository: GitHubRepository): Promise<string | undefined>;
}

export class StaticGitHubCredentialProvider implements GitHubCredentialProvider {
  private readonly secret: RuntimeSecret;

  constructor(token: string) { this.secret = new RuntimeSecret(token); }

  async getToken(_repository: GitHubRepository): Promise<string> {
    return this.secret.reveal();
  }
}

/** A secret that can cross an explicit runtime credential channel, but is safe to stringify or log. */
export class RuntimeSecret {
  constructor(private readonly value: string) {
    if (value.trim() === '') throw new GitHubRepositoryError('The GitHub credential must not be empty.');
  }

  reveal(): string {
    return this.value;
  }

  toString(): string {
    return '[REDACTED]';
  }

  toJSON(): string {
    return '[REDACTED]';
  }
}

export interface GitHubCloneCredential {
  kind: 'github-token';
  secret: RuntimeSecret;
}

const githubTokenPatterns = [
  /bearer\s+[A-Za-z0-9._~-]+/gi,
  /x-access-token:[^@\s]+/gi,
  /gh[pousr]_[A-Za-z0-9_]+/g,
  /github_pat_[A-Za-z0-9_]+/g,
];

export function redactGitHubSecrets(value: string, secrets: readonly RuntimeSecret[] = []): string {
  let redacted = value;
  for (const secret of secrets) redacted = redacted.split(secret.reveal()).join('[REDACTED]');
  for (const pattern of githubTokenPatterns) redacted = redacted.replace(pattern, '[REDACTED]');
  return redacted;
}
