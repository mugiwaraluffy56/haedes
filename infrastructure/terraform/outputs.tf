output "aws_region" {
  description = "The single region used by this environment."
  value       = var.aws_region
}

output "vpc_id" {
  description = "Environment VPC ID."
  value       = module.networking.vpc_id
}

output "public_subnet_ids" {
  description = "Public subnet IDs for future load balancer resources."
  value       = module.networking.public_subnet_ids
}

output "private_subnet_ids" {
  description = "Private subnet IDs for control-plane and sandbox tasks."
  value       = module.networking.private_subnet_ids
}

output "network_control_plane_security_group_id" {
  description = "Foundation security group reserved for control-plane network dependencies."
  value       = module.networking.control_plane_security_group_id
}

output "network_sandbox_security_group_id" {
  description = "Foundation security group reserved for sandbox network dependencies."
  value       = module.networking.sandbox_security_group_id
}

output "control_plane_repository_url" {
  description = "ECR repository URL for the control-plane image."
  value       = module.ecr.control_plane_repository_url
}

output "sandbox_repository_url" {
  description = "ECR repository URL for the sandbox image."
  value       = module.ecr.sandbox_repository_url
}

output "control_plane_image_uri" {
  description = "Explicit control-plane image input reserved for ECS composition."
  value       = var.control_plane_image_uri
}

output "sandbox_image_uri" {
  description = "Explicit sandbox image input reserved for ECS composition."
  value       = var.sandbox_image_uri
}

output "snapshot_bucket" {
  description = "Private, encrypted S3 bucket for workspace snapshots."
  value       = module.snapshots.bucket_name
}

output "metadata_table" {
  description = "DynamoDB table for sandbox metadata."
  value       = module.metadata.table_name
}

output "control_plane_log_group" {
  description = "CloudWatch log group for the control plane."
  value       = module.logs.control_plane_log_group_name
}

output "sandbox_log_group" {
  description = "CloudWatch log group for sandbox runtime tasks."
  value       = module.logs.sandbox_log_group_name
}

output "ecs_cluster_arn" {
  description = "ECS cluster ARN for sandbox task launches."
  value       = module.ecs.cluster_arn
}

output "control_plane_service_name" {
  description = "ECS control-plane service name."
  value       = module.control_plane.service_name
}

output "control_plane_alb_dns_name" {
  description = "Public ALB DNS name for the control-plane API."
  value       = module.control_plane.alb_dns_name
}

output "sandbox_task_definition_arn" {
  description = "Private per-sandbox task definition ARN."
  value       = module.sandbox.task_definition_arn
}

output "control_plane_security_group_id" {
  description = "Control-plane service security group ID."
  value       = module.control_plane.service_security_group_id
}

output "sandbox_security_group_id" {
  description = "Sandbox security group allowing runtime traffic only from the control-plane service."
  value       = module.control_plane.sandbox_security_group_id
}

output "sandbox_execution_role_arn" {
  description = "ECS execution role used to pull/log sandbox tasks; not exposed to containers."
  value       = module.iam.sandbox_execution_role_arn
}
