output "ec2_public_ip" {
  description = "IP pública de la instancia EC2"
  value       = module.ec2.public_ip
}

output "ecr_repository_url" {
  description = "URL del repositorio ECR"
  value       = module.ecr.repository_url
}

output "s3_bucket_name" {
  description = "Nombre del bucket S3"
  value       = module.s3.bucket_name
}

output "ssh_private_key" {
  description = "Clave privada SSH generada para acceder a la instancia"
  value       = tls_private_key.ssh_key.private_key_pem
  sensitive   = true
}

output "ssm_association_id" {
  description = "ID de la asociación SSM con EC2"
  value       = module.ssm.ssm_association_id
}

output "ec2_instance_id" {
  description = "ID de la instancia EC2"
  value       = module.ec2.instance_id
}

output "rds_endpoint" {
  description = "Endpoint del RDS PostgreSQL"
  value       = module.rds.db_endpoint
}

output "rds_address" {
  description = "Dirección del RDS PostgreSQL (sin puerto)"
  value       = module.rds.db_address
}

output "rds_port" {
  description = "Puerto del RDS PostgreSQL"
  value       = module.rds.db_port
}

output "rds_db_name" {
  description = "Nombre de la base de datos"
  value       = module.rds.db_name
}

output "rds_username" {
  description = "Usuario de la base de datos"
  value       = module.rds.db_username
  sensitive   = true
}

output "rds_password" {
  description = "Contraseña de la base de datos (solo si se generó automáticamente)"
  value       = var.db_password != "" ? null : random_password.db_password.result
  sensitive   = true
}

output "ses_domain_verified" {
  description = "Estado de verificacion del dominio en SES (true cuando Route 53 propago los TXT)"
  value       = aws_ses_domain_identity_verification.exwood.id != "" ? "verified" : "pending"
}

output "ses_dkim_tokens" {
  description = "Tokens DKIM generados por SES (usados en los CNAME del dominio)"
  value       = aws_ses_domain_dkim.exwood.dkim_tokens
}
