variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "retention_days" {
  type = number

  validation {
    condition     = var.retention_days >= 1 && var.retention_days <= 3653
    error_message = "retention_days must be between 1 and 3653."
  }
}

variable "tags" {
  type = map(string)
}
