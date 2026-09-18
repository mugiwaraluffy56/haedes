output "control_plane_log_group_name" {
  value = aws_cloudwatch_log_group.control_plane.name
}

output "sandbox_log_group_name" {
  value = aws_cloudwatch_log_group.sandbox.name
}

output "log_group_arns" {
  value = [aws_cloudwatch_log_group.control_plane.arn, aws_cloudwatch_log_group.sandbox.arn]
}
