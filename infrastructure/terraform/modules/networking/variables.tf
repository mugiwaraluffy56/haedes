variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "vpc_cidr" {
  type = string
}

variable "availability_zones" {
  type = list(string)

  validation {
    condition     = length(var.availability_zones) == 2
    error_message = "The networking module requires two availability zones."
  }
}

variable "public_subnet_cidrs" {
  type = list(string)

  validation {
    condition     = length(var.public_subnet_cidrs) == 2
    error_message = "The networking module requires two public subnet CIDRs."
  }
}

variable "private_subnet_cidrs" {
  type = list(string)

  validation {
    condition     = length(var.private_subnet_cidrs) == 2
    error_message = "The networking module requires two private subnet CIDRs."
  }
}

variable "enable_nat_gateway" {
  type    = bool
  default = true
}

variable "tags" {
  type = map(string)
}
