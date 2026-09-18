output "control_plane_repository_url" {
  value = aws_ecr_repository.images["control_plane"].repository_url
}

output "sandbox_repository_url" {
  value = aws_ecr_repository.images["sandbox"].repository_url
}
