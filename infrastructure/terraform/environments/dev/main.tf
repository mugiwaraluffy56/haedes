module "platform" {
  source = "../.."

  aws_region                  = var.aws_region
  project_name                = var.project_name
  environment                 = "dev"
  availability_zones          = var.availability_zones
  vpc_cidr                    = var.vpc_cidr
  public_subnet_cidrs         = var.public_subnet_cidrs
  private_subnet_cidrs        = var.private_subnet_cidrs
  enable_nat_gateway          = var.enable_nat_gateway
  snapshot_retention_days     = var.snapshot_retention_days
  log_retention_days          = var.log_retention_days
  control_plane_image_uri     = var.control_plane_image_uri
  sandbox_image_uri           = var.sandbox_image_uri
  control_plane_cpu           = var.control_plane_cpu
  control_plane_memory        = var.control_plane_memory
  sandbox_cpu                 = var.sandbox_cpu
  sandbox_memory              = var.sandbox_memory
  control_plane_desired_count = var.control_plane_desired_count
  tags                        = var.tags
}
