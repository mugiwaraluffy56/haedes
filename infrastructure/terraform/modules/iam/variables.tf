variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "snapshot_bucket_arn" {
  type = string
}

variable "metadata_table_arn" {
  type = string
}

variable "tags" {
  type = map(string)
}
