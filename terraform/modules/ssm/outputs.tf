output "ssm_association_id" {
  description = "ID de la asociación SSM con la instancia"
  value       = aws_ssm_association.ssm_agent.id
}
