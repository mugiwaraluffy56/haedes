# AWS development smoke test

The control-plane ECS task runs in AWS mode when Terraform sets
`HAEDES_AWS_ENABLED=true`. In that mode it launches sandbox tasks through ECS,
stores sandbox and snapshot metadata in DynamoDB, and stores workspace archives
in the encrypted S3 snapshot bucket.

After applying the development Terraform environment and deploying both
images, run:

```sh
HAEDES_API_BASE_URL=http://<control-plane-alb> \
HAEDES_API_KEY=<api-key> \
./scripts/smoke-aws.sh
```

The script verifies health, sandbox creation and cleanup, command execution,
file write/read, and snapshot creation. It deletes the sandbox on exit, even
when an assertion fails. AWS credentials are not passed into sandbox tasks;
only the control-plane ECS task role can call the AWS adapters.
