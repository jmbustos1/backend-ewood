variable "db_name" {
  description = "Nombre de la base de datos"
  type        = string
  default     = "ewood_db"
}

variable "db_username" {
  description = "Usuario administrador de la base de datos"
  type        = string
  default     = "ewood_admin"
}

variable "db_password" {
  description = "Contraseña del usuario administrador"
  type        = string
  sensitive   = true
}

variable "db_instance_class" {
  description = "Clase de instancia RDS"
  type        = string
  default     = "db.t3.micro" # Instancia pequeña para pruebas
}

variable "allocated_storage" {
  description = "Almacenamiento asignado en GB"
  type        = number
  default     = 20
}

variable "vpc_id" {
  description = "ID de la VPC"
  type        = string
}

variable "subnet_ids" {
  description = "IDs de las subnets para el RDS (al menos 2 en diferentes AZs)"
  type        = list(string)
}

variable "security_group_id" {
  description = "ID del security group de EC2 para permitir conexiones"
  type        = string
}

variable "aws_region" {
  description = "Región de AWS"
  type        = string
  default     = "us-east-1"
}

# ⚠️ VARIABLE DE DESARROLLO - ELIMINAR EN PRODUCCIÓN ⚠️
variable "allow_public_access" {
  description = "⚠️ SOLO PARA DESARROLLO: Permite acceso al RDS desde cualquier IP (0.0.0.0/0). DEBE SER false EN PRODUCCIÓN"
  type        = bool
  default     = true # ⚠️ CAMBIAR A false EN PRODUCCIÓN
}

