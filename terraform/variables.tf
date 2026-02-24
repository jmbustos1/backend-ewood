variable "aws_region" {
  description = "Región de AWS donde se desplegará la infraestructura"
  type        = string
  default     = "us-east-1"
}

variable "ecr_repository_name" {
  description = "Nombre del repositorio ECR"
  type        = string
  default     = "ewood-backend"
}

variable "s3_bucket_name" {
  description = "Nombre del bucket S3 para almacenar docker-compose y otros archivos"
  type        = string
  default     = "ewood-docker-compose-files"
}

variable "instance_type" {
  description = "Tipo de instancia EC2"
  type        = string
  default     = "t3a.nano"
}

variable "ami_id" {
  description = "AMI de la instancia EC2"
  type        = string
  default     = "ami-08c40ec9ead489470" # Ubuntu 22.04 en us-east-1
}

variable "vpc_id" {
  description = "ID de la VPC donde se desplegará EC2"
  type        = string
}

variable "subnet_ids" {
  description = "IDs de las subnets para RDS (al menos 2 en diferentes AZs). Si no se proporciona, se obtendrán automáticamente de la VPC"
  type        = list(string)
  default     = []
}

variable "db_password" {
  description = "Contraseña para el usuario administrador de RDS PostgreSQL"
  type        = string
  sensitive   = true
  default     = "" # Se puede generar automáticamente si está vacío
}

variable "db_username" {
  description = "Usuario administrador de RDS PostgreSQL"
  type        = string
  default     = "ewood_admin"
}

variable "db_instance_class" {
  description = "Clase de instancia RDS (db.t3.micro para pruebas, db.t3.small para producción)"
  type        = string
  default     = "db.t3.micro"
}
