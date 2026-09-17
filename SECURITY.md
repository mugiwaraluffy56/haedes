# Security policy

haedes runs agent-generated commands on temporary AWS computers. Treat every command, repository, and workspace as untrusted input.

The control plane owns AWS credentials and lifecycle state. Sandbox tasks must never receive AWS credentials. Runtime access uses a short-lived sandbox-scoped token, and `/workspace` is the only user filesystem root.

Please do not report security issues in a public issue. Contact the repository maintainers privately with a reproduction, affected boundary, and suggested mitigation.
