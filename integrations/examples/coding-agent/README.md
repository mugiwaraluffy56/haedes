# Coding-agent sandbox example

This scripted journey demonstrates the host boundary that a real coding agent
would consume. It creates a sandbox for a GitHub repository, runs the failing
fixture test, reads and edits the authentication source, reruns the test,
saves a snapshot, destroys the original sandbox, restores the snapshot into a
new sandbox, reruns the tests, and destroys the restored sandbox.

The deterministic edit is only a compatibility example; there is no model
planner or vendor-specific agent loop here. Configure `HAEDES_API_URL`,
`HAEDES_API_KEY`, and `HAEDES_REPOSITORY_URL` before running it. Set
`HAEDES_REPOSITORY_PATH` or `HAEDES_REPOSITORY_REF` when the repository uses
different values.
