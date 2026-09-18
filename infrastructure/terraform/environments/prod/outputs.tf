output "vpc_id" {
  value = module.platform.vpc_id
}

output "private_subnet_ids" {
  value = module.platform.private_subnet_ids
}

output "snapshot_bucket" {
  value = module.platform.snapshot_bucket
}

output "metadata_table" {
  value = module.platform.metadata_table
}

output "snapshot_table" {
  value = module.platform.snapshot_table
}

output "ecs_cluster_arn" {
  value = module.platform.ecs_cluster_arn
}

output "control_plane_service_name" {
  value = module.platform.control_plane_service_name
}

output "control_plane_alb_dns_name" {
  value = module.platform.control_plane_alb_dns_name
}

output "control_plane_security_group_id" {
  value = module.platform.control_plane_security_group_id
}

output "sandbox_security_group_id" {
  value = module.platform.sandbox_security_group_id
}

output "sandbox_task_definition_arn" {
  value = module.platform.sandbox_task_definition_arn
}
