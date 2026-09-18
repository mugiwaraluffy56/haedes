export const githubRepositoryInvalidCode = 'github_repository_invalid' as const;

export class GitHubRepositoryError extends Error {
  readonly code = githubRepositoryInvalidCode;

  constructor(message: string) {
    super(message);
    this.name = 'GitHubRepositoryError';
  }
}

export interface GitHubRepository {
  owner: string;
  name: string;
  url: string;
  cloneUrl: string;
}

const repositoryNamePattern = /^[A-Za-z0-9](?:[A-Za-z0-9._-]{0,98}[A-Za-z0-9])?$/;

function invalid(message: string): never {
  throw new GitHubRepositoryError(message);
}

function validatePart(value: string, label: 'owner' | 'repository'): string {
  if (!repositoryNamePattern.test(value) || value.length > 100 || value === '.' || value === '..') {
    invalid(`The GitHub ${label} name is invalid.`);
  }
  return value;
}

function makeRepository(owner: string, name: string): GitHubRepository {
  const validOwner = validatePart(owner, 'owner');
  const validName = validatePart(name, 'repository');
  const url = `https://github.com/${validOwner}/${validName}`;
  return { owner: validOwner, name: validName, url, cloneUrl: `${url}.git` };
}

function parseShorthand(value: string): GitHubRepository {
  const withoutGitSuffix = value.endsWith('.git') ? value.slice(0, -4) : value;
  const parts = withoutGitSuffix.split('/');
  if (parts.length !== 2 || parts.some((part) => part.length === 0)) {
    invalid('Use a GitHub HTTPS URL or owner/repository shorthand.');
  }
  return makeRepository(parts[0], parts[1]);
}

function parseHttpsUrl(value: string): GitHubRepository {
  if (!value.startsWith('https://') || /\/(?:\.{1,2})(?:\/|$)/.test(value) || /%2f|%2e/i.test(value)) {
    invalid('Only HTTPS GitHub repository URLs are supported.');
  }

  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    invalid('The GitHub repository URL is malformed.');
  }

  if (parsed.protocol !== 'https:' || parsed.hostname.toLowerCase() !== 'github.com' || parsed.port || parsed.username || parsed.password) {
    invalid('Only HTTPS GitHub repository URLs are supported.');
  }
  if (parsed.search || parsed.hash || parsed.pathname.includes('//')) {
    invalid('The GitHub repository URL must not contain a query, fragment, or repeated path separator.');
  }

  const parts = parsed.pathname.slice(1).split('/');
  if (parts.at(-1) === '') parts.pop();
  if (parts.length !== 2 || parts.some((part) => part.length === 0)) {
    invalid('The GitHub repository URL must contain exactly one owner and repository.');
  }
  return makeRepository(parts[0], parts[1].endsWith('.git') ? parts[1].slice(0, -4) : parts[1]);
}

export function parseRepositoryUrl(input: string): GitHubRepository {
  if (typeof input !== 'string' || input.length === 0 || input.trim() !== input || /[\u0000-\u001f\u007f\s]/.test(input)) {
    invalid('The GitHub repository reference is invalid.');
  }
  if (input.includes('://')) return parseHttpsUrl(input);
  return parseShorthand(input);
}

export function normalizeRepositoryUrl(input: string): string {
  return parseRepositoryUrl(input).url;
}
