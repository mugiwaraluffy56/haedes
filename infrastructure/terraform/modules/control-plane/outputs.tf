output "service_name" {
  value = aws_ecs_service.control_plane.name
}

output "task_definition_arn" {
  value = aws_ecs_task_definition.control_plane.arn
}

output "alb_dns_name" {
  value = aws_lb.public.dns_name
}

output "service_security_group_id" {
  value = aws_security_group.service.id
}

output "alb_security_group_id" {
  value = aws_security_group.alb.id
}

output "sandbox_security_group_id" {
  value = aws_security_group.sandbox.id
}
