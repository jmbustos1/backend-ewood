variable "ami_id" {
  description = "ID de la AMI a utilizar para la instancia EC2"
  type        = string
}

variable "instance_type" {
  description = "Tipo de instancia EC2"
  type        = string
}

variable "iam_instance_profile" {
  description = "Perfil IAM para la instancia"
  type        = string
}

variable "security_group" {
  description = "ID del Security Group para EC2"
  type        = string
}

variable "user_data" {
  description = "Script de inicialización para EC2"
  type        = string
}

variable "key_name" {
  description = "Nombre del par de claves SSH para acceder a la instancia"
  type        = string
}
