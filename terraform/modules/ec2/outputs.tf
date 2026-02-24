output "public_ip" {
  description = "IP pública de la instancia EC2"
  value       = aws_instance.tcp_server.public_ip
}

output "instance_id" {
  description = "ID de la instancia EC2"
  value       = aws_instance.tcp_server.id
}
