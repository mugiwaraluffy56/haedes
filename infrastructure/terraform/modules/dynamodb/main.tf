resource "aws_dynamodb_table" "metadata" {
  name         = "${var.project_name}-${var.environment}-metadata"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "sandboxId"

  attribute {
    name = "sandboxId"
    type = "S"
  }

  attribute {
    name = "ownerId"
    type = "S"
  }

  attribute {
    name = "createdAt"
    type = "S"
  }

  global_secondary_index {
    name            = "owner-createdAt-index"
    hash_key        = "ownerId"
    range_key       = "createdAt"
    projection_type = "ALL"
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-metadata" })
}
