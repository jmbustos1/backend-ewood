output "instance_profile_name" {
  description = "Nombre del perfil de instancia IAM para EC2"
  value       = aws_iam_instance_profile.ec2_instance_profile.name
}
