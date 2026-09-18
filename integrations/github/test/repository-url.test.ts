import assert from 'node:assert/strict';
import { createCloneRequest } from '../src/clone-request.js';
import { redactGitHubSecrets, RuntimeSecret, StaticGitHubCredentialProvider } from '../src/credentials.js';
import { GitHubRepositoryError, normalizeRepositoryUrl, parseRepositoryUrl } from '../src/repository-url.js';

const urlRepository = parseRepositoryUrl('https://github.com/haedes-dev/sandbox.git/');
assert.deepEqual(urlRepository, {
  owner: 'haedes-dev',
  name: 'sandbox',
  url: 'https://github.com/haedes-dev/sandbox',
  cloneUrl: 'https://github.com/haedes-dev/sandbox.git',
});
assert.deepEqual(parseRepositoryUrl('haedes-dev/sandbox'), urlRepository);
assert.equal(normalizeRepositoryUrl('haedes-dev/sandbox.git'), 'https://github.com/haedes-dev/sandbox');

for (const value of [
  'http://github.com/haedes-dev/sandbox',
  'https://gitlab.com/haedes-dev/sandbox',
  'https://user:password@github.com/haedes-dev/sandbox',
  'https://github.com/haedes-dev/sandbox/issues',
  'https://github.com/haedes-dev//sandbox',
  'https://github.com/haedes-dev/sandbox?token=secret',
  'github.com/haedes-dev/sandbox',
  'haedes-dev',
  'haedes-dev/sandbox/extra',
  'haedes-dev/../sandbox',
]) {
  assert.throws(() => parseRepositoryUrl(value), (error: unknown) => {
    if (!(error instanceof GitHubRepositoryError)) return false;
    assert.equal(error.code, 'github_repository_invalid');
    return true;
  });
}

const token = 'ghp_example_secret_123';
const request = await createCloneRequest({
  repository: 'haedes-dev/sandbox',
  destination: '/workspace/sandbox',
  credentials: new StaticGitHubCredentialProvider(token),
});
assert.deepEqual(request.command, ['git', 'clone', '--no-tags', 'https://github.com/haedes-dev/sandbox.git', '/workspace/sandbox']);
assert.equal(request.credential?.secret.reveal(), token);
assert.ok(!request.displayCommand.includes(token));
assert.ok(!JSON.stringify(request).includes(token));
assert.ok(!JSON.stringify(new StaticGitHubCredentialProvider(token)).includes(token));

await assert.rejects(createCloneRequest({ repository: 'haedes-dev/sandbox', destination: '/workspace/../outside' }), (error: unknown) => {
  assert.ok(error instanceof GitHubRepositoryError);
  assert.equal(error.code, 'github_repository_invalid');
  assert.match(error.message, /safe path/);
  return true;
});
assert.equal(new RuntimeSecret(token).toString(), '[REDACTED]');
assert.equal(redactGitHubSecrets(`clone Authorization: Bearer ${token}`), 'clone Authorization: [REDACTED]');
assert.equal(redactGitHubSecrets('remote x-access-token:ghp_example_secret_123@github.com'), 'remote [REDACTED]@github.com');
console.log('GitHub repository checks passed');
