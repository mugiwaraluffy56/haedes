output "vpc_id" {
  value = aws_vpc.main.id
}

output "public_subnet_ids" {
  value = [for key in sort(keys(aws_subnet.public)) : aws_subnet.public[key].id]
}

output "private_subnet_ids" {
  value = [for key in sort(keys(aws_subnet.private)) : aws_subnet.private[key].id]
}

output "control_plane_security_group_id" {
  value = aws_security_group.control_plane.id
}

output "sandbox_security_group_id" {
  value = aws_security_group.sandbox.id
}
