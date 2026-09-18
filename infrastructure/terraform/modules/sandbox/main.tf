resource "aws_ecs_task_definition" "sandbox" {
  family                   = "${var.project_name}-${var.environment}-sandbox"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(var.cpu)
  memory                   = tostring(var.memory)
  execution_role_arn       = var.execution_role_arn

  container_definitions = jsonencode([{
    name                   = var.runtime_container_name
    image                  = var.image_uri
    cpu                    = var.cpu
    memory                 = var.memory
    essential              = true
    user                   = "10001"
    readonlyRootFilesystem = true
    portMappings = [{
      containerPort = 8080
      hostPort      = 8080
      protocol      = "tcp"
    }]
    environment = [
      { name = "HAEDES_RUNTIME_BIND", value = "0.0.0.0:8080" },
      { name = "HAEDES_RUNTIME_WORKSPACE", value = "/workspace" }
    ]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = var.log_group_name
        "awslogs-region"        = "${data.aws_region.current.name}"
        "awslogs-stream-prefix" = "sandbox"
      }
    }
  }])

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-sandbox" })
}

data "aws_region" "current" {}
