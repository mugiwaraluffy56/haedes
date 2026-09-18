# Security tests

Boundary, isolation, and abuse-case tests belong here.

Run the repository-level boundary assertions with:

```sh
pnpm test:security
```

The runtime and control-plane suites additionally cover authenticated access,
owner scoping, invalid runtime tokens, workspace traversal and symlink escapes,
unsafe snapshot archives, bounded file/output bodies, timeouts, descendant
process termination, and credential override rejection. The boundary test
keeps the sandbox image and ECS task definition from accidentally acquiring
AWS credentials, host mounts, a Docker socket, public runtime ingress, or a
task role.
