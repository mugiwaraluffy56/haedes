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
  default = "10.42.0.0/16"
}

variable "public_subnet_cidrs" {
  type    = list(string)
  default = ["10.42.0.0/20", "10.42.16.0/20"]
}

variable "private_subnet_cidrs" {
  type    = list(string)
  default = ["10.42.32.0/20", "10.42.48.0/20"]
}

variable "enable_nat_gateway" {
  type    = bool
  default = true
}

variable "snapshot_retention_days" {
  type    = number
  default = 30
}

variable "log_retention_days" {
  type    = number
  default = 30
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
    CostCenter = "hackathon"
  }
}

variable "control_plane_cpu" {
  type    = number
  default = 512
}

variable "control_plane_memory" {
  type    = number
  default = 1024
}

variable "sandbox_cpu" {
  type    = number
  default = 1024
}

variable "sandbox_memory" {
  type    = number
  default = 2048
}

variable "control_plane_desired_count" {
  type    = number
  default = 1
}
