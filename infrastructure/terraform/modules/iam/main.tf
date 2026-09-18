data "aws_iam_policy_document" "ecs_tasks_trust" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "control_plane_execution" {
  name               = "${var.project_name}-${var.environment}-control-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
  tags               = merge(var.tags, { Name = "${var.project_name}-${var.environment}-control-execution" })
}

resource "aws_iam_role_policy_attachment" "control_plane_execution" {
  role       = aws_iam_role.control_plane_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role" "sandbox_execution" {
  name               = "${var.project_name}-${var.environment}-sandbox-execution"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
  tags               = merge(var.tags, { Name = "${var.project_name}-${var.environment}-sandbox-execution" })
}

resource "aws_iam_role_policy_attachment" "sandbox_execution" {
  role       = aws_iam_role.sandbox_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role" "control_plane_task" {
  name               = "${var.project_name}-${var.environment}-control-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_trust.json
  tags               = merge(var.tags, { Name = "${var.project_name}-${var.environment}-control-task" })
}

data "aws_iam_policy_document" "control_plane_task" {
  statement {
    sid       = "EcsSandboxLifecycle"
    effect    = "Allow"
    actions   = ["ecs:DescribeTaskDefinition", "ecs:DescribeTasks", "ecs:RunTask", "ecs:StopTask"]
    resources = ["*"]
  }

  statement {
    sid       = "PassSandboxExecutionRole"
    effect    = "Allow"
    actions   = ["iam:PassRole"]
    resources = [aws_iam_role.sandbox_execution.arn]
  }

  statement {
    sid       = "SnapshotObjects"
    effect    = "Allow"
    actions   = ["s3:DeleteObject", "s3:GetObject", "s3:PutObject"]
    resources = ["${var.snapshot_bucket_arn}/*"]
  }

  statement {
    sid       = "SnapshotBucketMetadata"
    effect    = "Allow"
    actions   = ["s3:GetBucketLocation", "s3:ListBucket"]
    resources = [var.snapshot_bucket_arn]
  }

  statement {
    sid       = "SandboxMetadata"
    effect    = "Allow"
    actions   = ["dynamodb:ConditionCheckItem", "dynamodb:DeleteItem", "dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:Query", "dynamodb:UpdateItem"]
    resources = [var.metadata_table_arn, "${var.metadata_table_arn}/index/*"]
  }

  statement {
    sid       = "ControlPlaneMetrics"
    effect    = "Allow"
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "control_plane_task" {
  name   = "${var.project_name}-${var.environment}-control-policy"
  role   = aws_iam_role.control_plane_task.id
  policy = data.aws_iam_policy_document.control_plane_task.json
}
