# GitHub fixer flow

The GitHub fixer story combines the provider-neutral session with the
`integrations/github` repository boundary and the authentication-bug fixture.
The real agent remains the decision-maker: it creates a sandbox, runs the
fixture test, edits the source, snapshots and destroys the first sandbox,
restores into a new sandbox, reruns the test, and destroys the restored one.

The executable host contract lives in
[`integrations/examples/coding-agent`](../../integrations/examples/coding-agent);
MCP wiring and real-client compatibility are separate integration tasks.
