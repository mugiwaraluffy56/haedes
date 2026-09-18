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
