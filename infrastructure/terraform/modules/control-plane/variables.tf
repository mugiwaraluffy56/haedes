variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "vpc_id" {
  type = string
}

variable "public_subnet_ids" {
  type = list(string)
}

variable "private_subnet_ids" {
  type = list(string)
}

variable "cluster_arn" {
  type = string
}

variable "sandbox_task_definition_arn" {
  type = string
}

variable "metadata_table_name" {
  type = string
}

variable "snapshot_table_name" {
  type = string
}

variable "snapshot_bucket_name" {
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

variable "desired_count" {
  type = number
}

variable "execution_role_arn" {
  type = string
}

variable "task_role_arn" {
  type = string
}

variable "log_group_name" {
  type = string
}

variable "tags" {
  type = map(string)
}
