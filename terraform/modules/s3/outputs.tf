output "bucket_name" {
  description = "Nombre del bucket S3"
  value       = aws_s3_bucket.docker_compose_bucket.id
}

output "bucket_arn" {
  description = "ARN del bucket S3"
  value       = aws_s3_bucket.docker_compose_bucket.arn
}
