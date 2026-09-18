# Basic sandbox agent example

This is a provider-neutral host example, not a model loop. It creates one
sandbox through the TypeScript SDK, streams a command, writes and reads a file,
saves a snapshot, prints sandbox/command/snapshot identities, and destroys the
sandbox in `finally`.

Configure `HAEDES_API_URL` and `HAEDES_API_KEY`, then run the example from a
host with access to the control plane. The host contract is supplied by
`@haedes/agent-adapter`; an actual agent remains responsible for deciding which
commands or file operations to request.
