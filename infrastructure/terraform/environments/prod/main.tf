module "platform" {
  source = "../.."

  aws_region              = var.aws_region
  project_name            = var.project_name
  environment             = "prod"
  availability_zones      = var.availability_zones
  vpc_cidr                = var.vpc_cidr
  public_subnet_cidrs     = var.public_subnet_cidrs
  private_subnet_cidrs    = var.private_subnet_cidrs
  enable_nat_gateway      = true
  snapshot_retention_days = var.snapshot_retention_days
  log_retention_days      = var.log_retention_days
  control_plane_image_uri = var.control_plane_image_uri
  sandbox_image_uri       = var.sandbox_image_uri
  tags                    = var.tags
}
