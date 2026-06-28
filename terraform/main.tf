terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = "exwood"
}

# Crear clave SSH si no existe
resource "tls_private_key" "ssh_key" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "aws_key_pair" "generated_key" {
  key_name   = "generated-key-golang-ewood"
  public_key = tls_private_key.ssh_key.public_key_openssh
}

# Módulo S3 para almacenar docker-compose
module "s3" {
  source      = "./modules/s3"
  bucket_name = var.s3_bucket_name
  aws_region  = var.aws_region
}

# Módulo ECR
module "ecr" {
  source          = "./modules/ecr"
  repository_name = var.ecr_repository_name
}

# Módulo IAM
module "iam" {
  source        = "./modules/iam"
  s3_bucket_arn = module.s3.bucket_arn
}

# Módulo Security Group
module "security_group" {
  source = "./modules/security-group"
  vpc_id = var.vpc_id
}

# Módulo EC2
module "ec2" {
  source               = "./modules/ec2"
  instance_type        = var.instance_type
  ami_id               = var.ami_id
  security_group       = module.security_group.security_group_id
  iam_instance_profile = module.iam.instance_profile_name
  key_name             = aws_key_pair.generated_key.key_name
  user_data            = file("./scripts/user_data.sh")
}

# Módulo AWS SSM
module "ssm" {
  source      = "./modules/ssm"
  instance_id = module.ec2.instance_id
}

# Obtener subnets de la VPC si no se proporcionaron
data "aws_subnets" "vpc_subnets" {
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }
}

# Validar que tengamos al menos 2 subnets en diferentes AZs para RDS
locals {
  # Obtener subnets disponibles
  available_subnets = length(var.subnet_ids) > 0 ? var.subnet_ids : data.aws_subnets.vpc_subnets.ids

  # RDS necesita al menos 2 subnets (tomar las primeras 2 si hay más)
  subnet_ids_for_rds = length(local.available_subnets) >= 2 ? slice(local.available_subnets, 0, 2) : local.available_subnets
}

# Módulo RDS PostgreSQL
module "rds" {
  source = "./modules/rds"

  db_name           = "ewood_db"
  db_username       = var.db_username
  db_password       = var.db_password != "" ? var.db_password : random_password.db_password.result
  db_instance_class = var.db_instance_class
  allocated_storage = 20

  vpc_id            = var.vpc_id
  subnet_ids        = local.subnet_ids_for_rds
  security_group_id = module.security_group.security_group_id
  aws_region        = var.aws_region

  # ⚠️ CONFIGURACIÓN DE DESARROLLO - CAMBIAR EN PRODUCCIÓN ⚠️
  # allow_public_access = true permite conexiones desde cualquier IP
  # EN PRODUCCIÓN: Cambiar a false para mayor seguridad
  allow_public_access = true # ⚠️ CAMBIAR A false EN PRODUCCIÓN
}

# Generar contraseña aleatoria si no se proporciona
resource "random_password" "db_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# ─── IP elastica para el backend ─────────────────────────────────────────────
# Mantiene la IP estable a traves de reboots / stop+start de la EC2.
# Sin EIP, la public IP de una EC2 cambia y rompe el A record + Caddy.
resource "aws_eip" "backend" {
  domain = "vpc"
  tags   = { Name = "ewood-backend-eip" }
}

resource "aws_eip_association" "backend" {
  instance_id   = module.ec2.instance_id
  allocation_id = aws_eip.backend.id
}

# ─── DNS — api.exwood.dev ────────────────────────────────────────────────────
# La Hosted Zone la crea Route 53 automaticamente al registrar el dominio.
# Solo agregamos el A record del subdominio apuntando al EIP.
data "aws_route53_zone" "main" {
  name         = "exwood.dev"
  private_zone = false
}

resource "aws_route53_record" "api" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "api.exwood.dev"
  type    = "A"
  ttl     = 300
  records = [aws_eip.backend.public_ip]
}
