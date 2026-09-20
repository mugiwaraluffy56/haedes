# AWS verification and cleanup runbook

Run these commands from the repository root after selecting the correct AWS
profile and region. Set the environment name explicitly; do not rely on the
default account.

```sh
export TF_DIR=infrastructure/terraform/environments/dev
export AWS_REGION=us-east-1
export CLUSTER_ARN="$(terraform -chdir="$TF_DIR" output -raw ecs_cluster_arn)"
export SERVICE_NAME="$(terraform -chdir="$TF_DIR" output -raw control_plane_service_name)"
export SNAPSHOT_BUCKET="$(terraform -chdir="$TF_DIR" output -raw snapshot_bucket)"
export METADATA_TABLE="$(terraform -chdir="$TF_DIR" output -raw metadata_table)"
export SNAPSHOT_TABLE="$(terraform -chdir="$TF_DIR" output -raw snapshot_table)"
export CONTROL_LOG_GROUP="$(terraform -chdir="$TF_DIR" output -raw control_plane_log_group)"
export SANDBOX_LOG_GROUP="$(terraform -chdir="$TF_DIR" output -raw sandbox_log_group)"
```

## Console verification checklist

In the AWS console, select the Terraform region and verify the following
resources before a demo:

- **ECS:** the `haedes-dev` cluster has Container Insights enabled; the
  control-plane service has the expected desired/running count; sandbox tasks
  are Fargate tasks in private subnets and carry a `haedes:sandbox-id` tag.
- **CloudWatch:** `/haedes/haedes/dev/control-plane` and
  `/haedes/haedes/dev/sandbox` exist with the configured retention; the newest
  streams contain the current request or task IDs.
- **S3:** the snapshot bucket has Block Public Access enabled, ownership set to
  BucketOwnerEnforced, default AES-256 encryption, versioning, and the
  `expire-snapshots` lifecycle rule. Do not use a public object URL.
- **DynamoDB:** the metadata and snapshot tables are `ACTIVE`, encrypted, have
  point-in-time recovery enabled, and show the expected TTL/index settings.
- **ECR/IAM:** both repositories are immutable and scan-on-push; the control
  plane task role has the expected ECS, S3, DynamoDB, and metrics permissions;
  the sandbox task has only its ECS execution role.

## ECS and task verification

```sh
aws ecs describe-clusters --clusters "$CLUSTER_ARN" \
  --query 'clusters[0].{status:status,insights:settings}'
aws ecs describe-services --cluster "$CLUSTER_ARN" --services "$SERVICE_NAME" \
  --query 'services[0].{status:status,desired:desiredCount,running:runningCount,pending:pendingCount,taskDefinition:taskDefinition}'
aws ecs list-tasks --cluster "$CLUSTER_ARN" --service-name "$SERVICE_NAME" \
  --desired-status RUNNING --query 'taskArns[]'
```

For a sandbox task, use the task ARN from the corresponding DynamoDB metadata
item or the ECS console's task tags:

```sh
aws ecs describe-tasks --cluster "$CLUSTER_ARN" --tasks <task-arn> --include TAGS \
  --query 'tasks[0].{lastStatus:lastStatus,health:healthStatus,privateIp:attachments[0].details[?name==`privateIPv4Address`].value,taskDefinition:taskDefinitionArn,tags:tags}'
```

Verify that the task is Fargate, has no public IP, runs the expected immutable
image, and carries the `haedes:sandbox-id` tag. Never include the runtime token
or full task metadata in a ticket or chat transcript.

## CloudWatch verification

```sh
aws logs describe-log-groups --log-group-name-prefix "/haedes/haedes/dev"
aws logs tail "$CONTROL_LOG_GROUP" --since 1h --follow
aws logs tail "$SANDBOX_LOG_GROUP" --since 1h --follow
```

Correlate the API request ID, sandbox ID, ECS task tag, and log stream
timestamp. Logs may contain command failures; they must not contain API keys,
AWS credentials, or runtime tokens.

## S3 verification

```sh
aws s3api get-public-access-block --bucket "$SNAPSHOT_BUCKET"
aws s3api get-bucket-ownership-controls --bucket "$SNAPSHOT_BUCKET"
aws s3api get-bucket-encryption --bucket "$SNAPSHOT_BUCKET"
aws s3api get-bucket-versioning --bucket "$SNAPSHOT_BUCKET"
aws s3api get-bucket-lifecycle-configuration --bucket "$SNAPSHOT_BUCKET"
aws s3api list-objects-v2 --bucket "$SNAPSHOT_BUCKET" --prefix "sandboxes/" \
  --query 'Contents[].{key:Key,size:Size,lastModified:LastModified}'
```

Snapshot keys have the form
`sandboxes/<sandbox-id>/snapshots/<snapshot-id>.tar.zst`. Confirm that an
archive exists after snapshot creation, that the bucket blocks public access,
and that the stored object uses server-side encryption.

## DynamoDB verification

Inspect state without projecting the sensitive task field:

```sh
aws dynamodb describe-table --table-name "$METADATA_TABLE" \
  --query 'Table.{status:TableStatus,pitr:PointInTimeRecoveryDescription,keys:KeySchema,indexes:GlobalSecondaryIndexes[].IndexName}'
aws dynamodb describe-time-to-live --table-name "$METADATA_TABLE"
aws dynamodb scan --table-name "$METADATA_TABLE" \
  --projection-expression 'sandboxId,#state,ownerId,createdAt,expiresAt' \
  --expression-attribute-names '{"#state":"state"}'
aws dynamodb describe-table --table-name "$SNAPSHOT_TABLE" \
  --query 'Table.{status:TableStatus,pitr:PointInTimeRecoveryDescription,indexes:GlobalSecondaryIndexes[].IndexName}'
aws dynamodb describe-time-to-live --table-name "$SNAPSHOT_TABLE"
```

Expected states follow the lifecycle contract. Expired records are removed by
DynamoDB TTL asynchronously, so TTL is not an immediate deletion guarantee.
Snapshot metadata must match the S3 archive checksum before restore.

## Orphan task handling

An orphan is a running ECS task with a `haedes:sandbox-id` tag that has no
corresponding non-terminal sandbox record, or a task whose record is already
`destroyed`. Confirm the mismatch using the ECS task tag and a projected
DynamoDB scan before stopping anything.

```sh
aws ecs list-tasks --cluster "$CLUSTER_ARN" --desired-status RUNNING \
  --query 'taskArns[]' --output text
aws ecs describe-tasks --cluster "$CLUSTER_ARN" --tasks <task-arn> --include TAGS \
  --query 'tasks[0].{status:lastStatus,stoppedReason:stoppedReason,tags:tags}'
```

For a confirmed orphan, stop the task and record the reason:

```sh
aws ecs stop-task --cluster "$CLUSTER_ARN" --task <task-arn> \
  --reason "haedes orphan cleanup"
```

If a sandbox record still exists, use the authenticated `DELETE /v1/sandboxes`
operation as the normal cleanup path. Do not manually delete DynamoDB records
or S3 objects until evidence has been retained and the cleanup owner approves.

## Development teardown

1. Destroy every active sandbox through the API and verify no tagged tasks
   remain.
2. Preserve required evidence, then remove development snapshot objects and
   all object versions. The bucket is versioned, so deleting only current keys
   is insufficient.
3. Empty development ECR repositories if Terraform reports images prevent
   repository deletion.
4. Review the plan and destroy only the named development workspace:

   ```sh
   terraform -chdir="$TF_DIR" plan -destroy -var-file=terraform.tfvars
   terraform -chdir="$TF_DIR" destroy -var-file=terraform.tfvars
   ```

Never destroy a production environment or erase incident evidence as part of
routine cleanup.
