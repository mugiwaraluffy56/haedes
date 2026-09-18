# GitHub integration boundary

This boundary owns GitHub repository reference validation and credential exchange.
It does not own sandbox lifecycle, call AWS, or expose credentials to command
arguments, logs, agents, or public API inputs.

`parseRepositoryUrl` accepts an HTTPS `github.com/owner/repository` URL or
`owner/repository` shorthand and returns a canonical URL plus clone URL. It
rejects non-GitHub hosts, non-HTTPS schemes, embedded credentials, extra path
segments, queries, fragments, and unsafe repository names with the typed
`github_repository_invalid` error.

`createCloneRequest` produces an argument-array command with no secret embedded
in it. An injected `GitHubCredentialProvider` can supply a token through the
separate `credential` channel. The `RuntimeSecret` wrapper redacts itself when
stringified; a runtime adapter may explicitly reveal it only while setting up a
short-lived clone credential mechanism.

Use `redactGitHubSecrets` before forwarding command or event text to logs or
agent-visible output.

```ts
const request = await createCloneRequest({
  repository: 'octo/project',
  destination: '/workspace/project',
  credentials: credentialProvider,
});

// Pass request.command to the runtime and request.credential through its
// secret channel. Never concatenate request.credential into the command.
```
