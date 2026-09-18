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

output "control_plane_security_group_id" {
  description = "Security group for the public control-plane service."
  value       = module.networking.control_plane_security_group_id
}

output "sandbox_security_group_id" {
  description = "Security group allowing runtime traffic only from control plane."
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
