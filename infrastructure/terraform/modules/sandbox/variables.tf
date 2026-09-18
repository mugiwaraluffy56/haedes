variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "image_uri" {
  type = string
}

variable "cpu" {
  type = number
}

variable "memory" {
  type = number
}

variable "execution_role_arn" {
  type = string
}

variable "log_group_name" {
  type = string
}

variable "runtime_container_name" {
  type = string
}

variable "tags" {
  type = map(string)
}
