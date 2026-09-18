variable "aws_region" {
  description = "The one AWS region used by this environment."
  type        = string
  default     = "us-east-1"

  validation {
    condition     = can(regex("^[a-z]{2}-[a-z]+-[0-9]+$", var.aws_region))
    error_message = "aws_region must be a single valid-looking AWS region name."
  }
}

variable "project_name" {
  description = "Short, lowercase name used in resource names."
  type        = string
  default     = "haedes"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,20}$", var.project_name))
    error_message = "project_name must be 3-21 lowercase letters, numbers, or hyphens."
  }
}

variable "environment" {
  description = "Deployment environment name."
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "prod"], var.environment)
    error_message = "environment must be dev or prod."
  }
}

variable "availability_zones" {
  description = "Exactly two availability zones in aws_region."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]

  validation {
    condition     = length(var.availability_zones) == 2 && var.availability_zones[0] != var.availability_zones[1]
    error_message = "Exactly two distinct availability zones are required."
  }
}

variable "vpc_cidr" {
  description = "CIDR range for the single environment VPC."
  type        = string
  default     = "10.42.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "Two public subnet CIDRs, one per availability zone."
  type        = list(string)
  default     = ["10.42.0.0/20", "10.42.16.0/20"]

  validation {
    condition     = length(var.public_subnet_cidrs) == 2
    error_message = "Exactly two public subnet CIDRs are required."
  }
}

variable "private_subnet_cidrs" {
  description = "Two private subnet CIDRs, one per availability zone."
  type        = list(string)
  default     = ["10.42.32.0/20", "10.42.48.0/20"]

  validation {
    condition     = length(var.private_subnet_cidrs) == 2
    error_message = "Exactly two private subnet CIDRs are required."
  }
}

variable "enable_nat_gateway" {
  description = "Create one NAT gateway for private subnet egress."
  type        = bool
  default     = true
}

variable "snapshot_retention_days" {
  description = "Retention period for snapshot objects and non-current versions."
  type        = number
  default     = 30

  validation {
    condition     = var.snapshot_retention_days >= 1 && var.snapshot_retention_days <= 3653
    error_message = "snapshot_retention_days must be between 1 and 3653."
  }
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days."
  type        = number
  default     = 30

  validation {
    condition     = var.log_retention_days >= 1 && var.log_retention_days <= 3653
    error_message = "log_retention_days must be between 1 and 3653."
  }
}

variable "control_plane_image_uri" {
  description = "ECR image URI reserved for the control-plane service."
  type        = string

  validation {
    condition     = trimspace(var.control_plane_image_uri) != "" && (var.environment != "prod" || can(regex("@sha256:[0-9a-f]{64}$", var.control_plane_image_uri)))
    error_message = "control_plane_image_uri must be non-empty; prod requires a sha256 digest."
  }
}

variable "sandbox_image_uri" {
  description = "ECR image URI reserved for the sandbox runtime task."
  type        = string

  validation {
    condition     = trimspace(var.sandbox_image_uri) != "" && (var.environment != "prod" || can(regex("@sha256:[0-9a-f]{64}$", var.sandbox_image_uri)))
    error_message = "sandbox_image_uri must be non-empty; prod requires a sha256 digest."
  }
}

variable "tags" {
  description = "Additional tags. Owner and CostCenter are required."
  type        = map(string)
  default = {
    Owner      = "platform"
    CostCenter = "hackathon"
  }

  validation {
    condition     = trimspace(lookup(var.tags, "Owner", "")) != "" && trimspace(lookup(var.tags, "CostCenter", "")) != ""
    error_message = "tags must include non-empty Owner and CostCenter values."
  }
}

variable "control_plane_cpu" {
  description = "Fargate CPU units for the control-plane service."
  type        = number
  default     = 512

  validation {
    condition     = var.control_plane_cpu >= 256 && var.control_plane_cpu <= 4096
    error_message = "control_plane_cpu must be between 256 and 4096 CPU units."
  }
}

variable "control_plane_memory" {
  description = "Fargate memory in MiB for the control-plane service."
  type        = number
  default     = 1024

  validation {
    condition     = var.control_plane_memory >= 512 && var.control_plane_memory <= 8192
    error_message = "control_plane_memory must be between 512 and 8192 MiB."
  }
}

variable "sandbox_cpu" {
  description = "Fargate CPU units for sandbox tasks."
  type        = number
  default     = 1024

  validation {
    condition     = var.sandbox_cpu >= 256 && var.sandbox_cpu <= 4096
    error_message = "sandbox_cpu must be between 256 and 4096 CPU units."
  }
}

variable "sandbox_memory" {
  description = "Fargate memory in MiB for sandbox tasks."
  type        = number
  default     = 2048

  validation {
    condition     = var.sandbox_memory >= 512 && var.sandbox_memory <= 16384
    error_message = "sandbox_memory must be between 512 and 16384 MiB."
  }
}

variable "control_plane_desired_count" {
  description = "Desired control-plane task count."
  type        = number
  default     = 1

  validation {
    condition     = var.control_plane_desired_count >= 1 && var.control_plane_desired_count <= 4
    error_message = "control_plane_desired_count must be between 1 and 4."
  }
}
