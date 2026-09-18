variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "project_name" {
  type    = string
  default = "haedes"
}

variable "availability_zones" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b"]
}

variable "vpc_cidr" {
  type    = string
  default = "10.52.0.0/16"
}

variable "public_subnet_cidrs" {
  type    = list(string)
  default = ["10.52.0.0/20", "10.52.16.0/20"]
}

variable "private_subnet_cidrs" {
  type    = list(string)
  default = ["10.52.32.0/20", "10.52.48.0/20"]
}

variable "snapshot_retention_days" {
  type    = number
  default = 365
}

variable "log_retention_days" {
  type    = number
  default = 90
}

variable "control_plane_image_uri" {
  type = string
}

variable "sandbox_image_uri" {
  type = string
}

variable "tags" {
  type = map(string)
  default = {
    Owner      = "platform"
    CostCenter = "production"
  }
}

variable "control_plane_cpu" {
  type    = number
  default = 1024
}

variable "control_plane_memory" {
  type    = number
  default = 2048
}

variable "sandbox_cpu" {
  type    = number
  default = 2048
}

variable "sandbox_memory" {
  type    = number
  default = 4096
}

variable "control_plane_desired_count" {
  type    = number
  default = 2
}
