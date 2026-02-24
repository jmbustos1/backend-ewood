output "db_endpoint" {
  description = "Endpoint del RDS (hostname)"
  value       = aws_db_instance.postgres.endpoint
}

output "db_address" {
  description = "Dirección del RDS (sin puerto)"
  value       = aws_db_instance.postgres.address
}

output "db_port" {
  description = "Puerto del RDS"
  value       = aws_db_instance.postgres.port
}

output "db_name" {
  description = "Nombre de la base de datos"
  value       = aws_db_instance.postgres.db_name
}

output "db_username" {
  description = "Usuario de la base de datos"
  value       = aws_db_instance.postgres.username
  sensitive   = true
}

output "security_group_id" {
  description = "ID del security group de RDS"
  value       = aws_security_group.rds.id
}

