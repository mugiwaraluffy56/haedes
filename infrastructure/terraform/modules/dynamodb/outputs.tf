output "table_name" {
  value = aws_dynamodb_table.metadata.name
}

output "table_arn" {
  value = aws_dynamodb_table.metadata.arn
}

output "snapshot_table_name" {
  value = aws_dynamodb_table.snapshots.name
}

output "snapshot_table_arn" {
  value = aws_dynamodb_table.snapshots.arn
}
