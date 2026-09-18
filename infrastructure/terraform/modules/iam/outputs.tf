output "control_plane_execution_role_arn" {
  value = aws_iam_role.control_plane_execution.arn
}

output "control_plane_task_role_arn" {
  value = aws_iam_role.control_plane_task.arn
}

output "sandbox_execution_role_arn" {
  value = aws_iam_role.sandbox_execution.arn
}
