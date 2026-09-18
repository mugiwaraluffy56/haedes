# Terraform boundary

Reusable AWS infrastructure modules and environment compositions belong here.

## Structure

The root module defines one-region shared foundation resources. The `dev` and
`prod` directories are syntactically valid compositions that pass their
environment-specific inputs to that root module.

```sh
cp infrastructure/terraform/environments/dev/terraform.tfvars.example /tmp/haedes-dev.tfvars
terraform -chdir=infrastructure/terraform/environments/dev init -backend=false
terraform -chdir=infrastructure/terraform/environments/dev fmt -check -recursive
terraform -chdir=infrastructure/terraform/environments/dev validate
terraform -chdir=infrastructure/terraform/environments/dev plan -var-file=/tmp/haedes-dev.tfvars
```

The foundation creates a two-AZ VPC with public and private subnets, one
optional NAT gateway, least-privilege control-plane/runtime security groups,
immutable scanned ECR repositories, a private encrypted/versioned snapshot
bucket, a point-in-time-recoverable DynamoDB metadata table, retained
CloudWatch log groups, an ALB-backed private control-plane Fargate service,
and a private per-sandbox Fargate task definition. Sandbox definitions use
`awsvpc`, disabled public IP assignment, UID `10001`, read-only roots, and no
task role; the control-plane role has only the ECS, S3, DynamoDB, CloudWatch,
and required `iam:PassRole` actions. It does not create or apply production
credentials.

`control_plane_image_uri` and `sandbox_image_uri` are required so image
selection stays explicit and task definitions cannot silently use a mutable or
empty image reference.
