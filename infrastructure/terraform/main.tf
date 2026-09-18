locals {
  common_tags = merge(var.tags, {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  })
}

module "networking" {
  source = "./modules/networking"

  project_name         = var.project_name
  environment          = var.environment
  vpc_cidr             = var.vpc_cidr
  availability_zones   = var.availability_zones
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
  enable_nat_gateway   = var.enable_nat_gateway
  tags                 = local.common_tags
}

module "ecr" {
  source = "./modules/ecr"

  project_name = var.project_name
  environment  = var.environment
  tags         = local.common_tags
}

module "snapshots" {
  source = "./modules/s3"

  project_name     = var.project_name
  environment      = var.environment
  retention_days   = var.snapshot_retention_days
  tags             = local.common_tags
}

module "metadata" {
  source = "./modules/dynamodb"

  project_name = var.project_name
  environment  = var.environment
  tags         = local.common_tags
}

module "logs" {
  source = "./modules/cloudwatch"

  project_name  = var.project_name
  environment   = var.environment
  retention_days = var.log_retention_days
  tags          = local.common_tags
}

module "ecs" {
  source = "./modules/ecs"

  project_name = var.project_name
  environment  = var.environment
  tags         = local.common_tags
}

module "iam" {
  source = "./modules/iam"

  project_name        = var.project_name
  environment         = var.environment
  snapshot_bucket_arn = module.snapshots.bucket_arn
  metadata_table_arn  = module.metadata.table_arn
  snapshot_table_arn  = module.metadata.snapshot_table_arn
  tags                = local.common_tags
}

module "sandbox" {
  source = "./modules/sandbox"

  project_name          = var.project_name
  environment           = var.environment
  image_uri             = var.sandbox_image_uri
  cpu                   = var.sandbox_cpu
  memory                = var.sandbox_memory
  execution_role_arn    = module.iam.sandbox_execution_role_arn
  log_group_name        = module.logs.sandbox_log_group_name
  runtime_container_name = "runtime"
  tags                  = local.common_tags
}

module "control_plane" {
  source = "./modules/control-plane"

  project_name                = var.project_name
  environment                 = var.environment
  aws_region                  = var.aws_region
  vpc_id                      = module.networking.vpc_id
  public_subnet_ids           = module.networking.public_subnet_ids
  private_subnet_ids          = module.networking.private_subnet_ids
  cluster_arn                 = module.ecs.cluster_arn
  sandbox_task_definition_arn = module.sandbox.task_definition_arn
  metadata_table_name         = module.metadata.table_name
  snapshot_table_name         = module.metadata.snapshot_table_name
  snapshot_bucket_name        = module.snapshots.bucket_name
  image_uri                   = var.control_plane_image_uri
  cpu                         = var.control_plane_cpu
  memory                      = var.control_plane_memory
  desired_count               = var.control_plane_desired_count
  execution_role_arn          = module.iam.control_plane_execution_role_arn
  task_role_arn               = module.iam.control_plane_task_role_arn
  log_group_name              = module.logs.control_plane_log_group_name
  tags                        = local.common_tags
}
