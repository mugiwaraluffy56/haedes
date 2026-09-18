import type { GitHubCredentialProvider, GitHubCloneCredential } from './credentials.js';
import { RuntimeSecret } from './credentials.js';
import { GitHubRepositoryError, parseRepositoryUrl, type GitHubRepository } from './repository-url.js';

export interface CloneRequestInput {
  repository: string | GitHubRepository;
  destination: string;
  credentials?: GitHubCredentialProvider;
}

export interface CloneRequest {
  repository: GitHubRepository;
  destination: string;
  command: readonly string[];
  displayCommand: string;
  /** Delivered through a runtime secret channel; it is never part of command arguments. */
  credential?: GitHubCloneCredential;
}

function validateDestination(destination: string): string {
  if (!/^\/workspace(?:\/[A-Za-z0-9._-]+)+$/.test(destination) || destination.includes('..')) {
    throw new GitHubRepositoryError('The clone destination must be a safe path below /workspace.');
  }
  return destination;
}

function shellQuote(argument: string): string {
  return `'${argument.replaceAll("'", "'\\''")}'`;
}

export async function createCloneRequest(input: CloneRequestInput): Promise<CloneRequest> {
  const repository = typeof input.repository === 'string'
    ? parseRepositoryUrl(input.repository)
    : parseRepositoryUrl(input.repository.url);
  const destination = validateDestination(input.destination);
  const credentialToken = input.credentials ? await input.credentials.getToken(repository) : undefined;
  const credential = credentialToken === undefined
    ? undefined
    : { kind: 'github-token' as const, secret: new RuntimeSecret(credentialToken) };
  const command = ['git', 'clone', '--no-tags', repository.cloneUrl, destination] as const;

  return {
    repository,
    destination,
    command,
    displayCommand: command.map(shellQuote).join(' '),
    credential,
  };
}
