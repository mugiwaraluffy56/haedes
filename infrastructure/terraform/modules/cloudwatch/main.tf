resource "aws_cloudwatch_log_group" "control_plane" {
  name              = "/haedes/${var.project_name}/${var.environment}/control-plane"
  retention_in_days = var.retention_days

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-control-plane-logs" })
}

resource "aws_cloudwatch_log_group" "sandbox" {
  name              = "/haedes/${var.project_name}/${var.environment}/sandbox"
  retention_in_days = var.retention_days

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-sandbox-logs" })
}
