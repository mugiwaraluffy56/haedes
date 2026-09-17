# Claude Code notes

Read [`AGENTS.md`](AGENTS.md) and the source-of-truth documents in [`docs/`](docs/) before making changes.

The repository is provider-neutral. Claude Code may be used as one client during the hackathon, but haedes must not depend on Claude, OpenAI, Gemini, or any single model provider.

Keep MCP thin: translate agent tool calls into the public sandbox contract and reuse existing lifecycle, authentication, error, and identity semantics.
