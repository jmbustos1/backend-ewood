output "repository_url" {
  description = "URL del repositorio ECR"
  value       = aws_ecr_repository.ec2-go.repository_url
}
