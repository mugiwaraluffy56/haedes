resource "aws_security_group" "alb" {
  name_prefix = "${var.project_name}-${var.environment}-alb-"
  description = "Public HTTP entrypoint for the control-plane API."
  vpc_id      = var.vpc_id

  ingress {
    description = "Public API"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-alb" })
}

resource "aws_security_group" "service" {
  name_prefix = "${var.project_name}-${var.environment}-control-service-"
  description = "Control-plane traffic only from the public ALB."
  vpc_id      = var.vpc_id

  ingress {
    description     = "ALB to control plane"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-control-service" })
}

resource "aws_security_group" "sandbox" {
  name_prefix = "${var.project_name}-${var.environment}-sandbox-runtime-"
  description = "Private runtime ingress only from the control-plane service."
  vpc_id      = var.vpc_id

  ingress {
    description     = "Control plane to sandbox runtime"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.service.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-sandbox-runtime" })
}

resource "aws_ecs_task_definition" "control_plane" {
  family                   = "${var.project_name}-${var.environment}-control-plane"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = tostring(var.cpu)
  memory                   = tostring(var.memory)
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn

  container_definitions = jsonencode([{
    name      = "control-plane"
    image     = var.image_uri
    cpu       = var.cpu
    memory    = var.memory
    essential = true
    readonlyRootFilesystem = true
    portMappings = [{
      containerPort = 8080
      hostPort      = 8080
      protocol      = "tcp"
    }]
    environment = [
      { name = "HAEDES_ENV", value = var.environment == "prod" ? "production" : "development" },
      { name = "HAEDES_AWS_REGION", value = var.aws_region },
      { name = "HAEDES_CONTROL_PLANE_BIND", value = "0.0.0.0:8080" }
    ]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = var.log_group_name
        "awslogs-region"        = var.aws_region
        "awslogs-stream-prefix" = "control-plane"
      }
    }
  }])

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-control-plane" })
}

resource "aws_lb" "public" {
  name               = substr("${var.project_name}-${var.environment}-api", 0, 32)
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = var.public_subnet_ids

  enable_deletion_protection = var.environment == "prod"
  tags                        = merge(var.tags, { Name = "${var.project_name}-${var.environment}-api" })
}

resource "aws_lb_target_group" "control_plane" {
  name        = substr("${var.project_name}-${var.environment}-api", 0, 32)
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = var.vpc_id

  health_check {
    enabled             = true
    path                = "/healthz"
    port                = "traffic-port"
    protocol            = "HTTP"
    healthy_threshold   = 2
    unhealthy_threshold = 3
    interval            = 30
    timeout             = 5
    matcher             = "200"
  }

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-api" })
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.public.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.control_plane.arn
  }
}

resource "aws_ecs_service" "control_plane" {
  name            = "${var.project_name}-${var.environment}-control-plane"
  cluster         = var.cluster_arn
  task_definition = aws_ecs_task_definition.control_plane.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 60

  network_configuration {
    subnets          = var.private_subnet_ids
    security_groups  = [aws_security_group.service.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.control_plane.arn
    container_name   = "control-plane"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.http]

  tags = merge(var.tags, { Name = "${var.project_name}-${var.environment}-control-plane" })
}
