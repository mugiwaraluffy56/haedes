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
