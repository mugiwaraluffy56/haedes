# AWS deployment and operations

This runbook describes the single-region development/prod-shaped deployment
defined in `infrastructure/terraform`. It is intentionally explicit: do not
apply Terraform until the AWS account, region, image digests, tags, and cleanup
owner are known.

## Resource inventory

| Resource | Purpose | Important properties |
| --- | --- | --- |
| VPC, public/private subnets, routes, NAT | Network placement and package/Git egress | Two AZs; ECS tasks use private subnets; NAT is optional for development |
| ALB and target group | Public control-plane entry point | `/healthz` health check; forwards port 80 to control plane port 8080 |
| ECS cluster | Task scheduler | Container Insights enabled |
| Control-plane ECS service | `/v1` API and AWS orchestration | Fargate, private subnets, read-only root filesystem |
| Sandbox task definition | One runtime per sandbox | Fargate, non-root runtime, private network, port 8080 only from control plane |
| ECR repositories | Control-plane and sandbox images | Immutable tags, scan-on-push, AES-256, lifecycle keeps newest 20 images |
| DynamoDB tables | Sandbox and snapshot metadata | On-demand billing, SSE, point-in-time recovery, TTL, owner/sandbox indexes |
| S3 bucket | Workspace snapshot archives | Private, Block Public Access, enforced ownership, SSE, versioning, lifecycle expiration |
| CloudWatch log groups | Control-plane and sandbox logs | Configurable retention, default 30 days |
| IAM roles/policy | ECS execution and control-plane task access | Sandbox execution role has no application AWS permissions |

The root outputs are the operational handoff: `control_plane_alb_dns_name`,
`ecs_cluster_arn`, `control_plane_service_name`, `snapshot_bucket`,
`metadata_table`, `snapshot_table`, both log groups, and the sandbox task
definition ARN.

## Deploy a development environment

Prerequisites: AWS credentials with approved Terraform permissions, Docker,
Terraform 1.9.8 or the repository-pinned version, AWS CLI, `jq`, and a
configured region. Use a dedicated AWS account or tagged development
environment.

1. Copy the variables and set image URIs in
   `infrastructure/terraform/environments/dev/terraform.tfvars`. Development
   may use tags; production image inputs must use `@sha256:<digest>`.

   ```sh
   cp infrastructure/terraform/environments/dev/terraform.tfvars.example \
      infrastructure/terraform/environments/dev/terraform.tfvars
   export AWS_REGION=us-east-1
   export AWS_ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
   ```

2. Initialize and validate the environment without applying it:

   ```sh
   terraform -chdir=infrastructure/terraform/environments/dev init
   terraform -chdir=infrastructure/terraform/environments/dev fmt -check -recursive
   terraform -chdir=infrastructure/terraform/environments/dev validate
   terraform -chdir=infrastructure/terraform/environments/dev plan \
     -var-file=terraform.tfvars
   ```

3. Create ECR, build the amd64 images, and push them. The sandbox image used
   by ECS is the development image produced by the repository script; the
   intermediate base image does not need its own ECR repository.

   ```sh
   terraform -chdir=infrastructure/terraform/environments/dev apply \
     -target=module.platform.module.ecr \
     -var-file=terraform.tfvars

   export CONTROL_REPO="$(terraform -chdir=infrastructure/terraform/environments/dev output -raw control_plane_repository_url)"
   export SANDBOX_REPO="$(terraform -chdir=infrastructure/terraform/environments/dev output -raw sandbox_repository_url)"
   aws ecr get-login-password --region "$AWS_REGION" | \
     docker login --username AWS --password-stdin "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

   IMAGE_TAG=dev ./scripts/build-images.sh
   docker build --platform linux/amd64 \
     --file infrastructure/docker/control-plane.Dockerfile \
     --tag "$CONTROL_REPO:dev" .
   docker tag haedes-sandbox-dev:dev "$SANDBOX_REPO:dev"
   docker push "$CONTROL_REPO:dev"
   docker push "$SANDBOX_REPO:dev"
   ```

4. Apply the full stack and wait for the health check:

   ```sh
   terraform -chdir=infrastructure/terraform/environments/dev apply \
     -var-file=terraform.tfvars
   export HAEDES_API_BASE_URL="http://$(terraform -chdir=infrastructure/terraform/environments/dev output -raw control_plane_alb_dns_name)"
   curl --fail "$HAEDES_API_BASE_URL/healthz"
   ```

   For production, replace mutable tags with image digests, review the plan,
   and require an approved change window. This repository's ALB listener is
   HTTP for the development shape; add TLS termination and an approved
   certificate before exposing a production deployment.

5. Run the opt-in API smoke test only after the control plane and images are
   healthy:

   ```sh
   AWS_INTEGRATION_TESTS=true \
   HAEDES_ENV=development \
   HAEDES_API_BASE_URL="$HAEDES_API_BASE_URL" \
   HAEDES_API_KEY="<development-api-key>" \
   ./scripts/smoke-aws.sh
   ```

The smoke script removes its sandbox on exit. It does not replace the full
coding-agent demo; use [the demo runbook](../coding-agent-demo.md) for that.

## Verification and cleanup

Use [operations.md](operations.md) for console/CLI verification, orphan task
handling, and safe development teardown. Never run `terraform destroy` for a
production environment or while an incident is under investigation.
