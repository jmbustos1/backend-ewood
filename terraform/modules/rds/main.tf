# Security Group para RDS
resource "aws_security_group" "rds" {
  name        = "rds-ewood-postgres"
  description = "Security group para RDS PostgreSQL"
  vpc_id      = var.vpc_id

  # Permitir conexiones PostgreSQL desde el security group de EC2
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [var.security_group_id]
    description     = "PostgreSQL from EC2"
  }

  # REGLA TEMPORAL PARA DESARROLLO - ELIMINAR EN PRODUCCION
  # Esta regla permite conexiones desde cualquier IP (0.0.0.0/0)
  # SOLO debe estar activa durante desarrollo/testing
  # EN PRODUCCION: Eliminar este bloque o establecer allow_public_access = false
  dynamic "ingress" {
    for_each = var.allow_public_access ? [1] : []
    content {
      from_port   = 5432
      to_port     = 5432
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      description = "DEVELOPMENT ONLY: PostgreSQL from anywhere - REMOVE IN PRODUCTION"
    }
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound traffic"
  }

  tags = {
    Name = "rds-ewood-postgres-sg"
  }
}

# Subnet Group para RDS (requiere al menos 2 subnets en diferentes AZs)
resource "aws_db_subnet_group" "rds" {
  name       = "ewood-rds-subnet-group"
  subnet_ids = var.subnet_ids

  tags = {
    Name = "ewood-rds-subnet-group"
  }
}

# Instancia RDS PostgreSQL
resource "aws_db_instance" "postgres" {
  identifier = "ewood-postgres"

  engine         = "postgres"
  engine_version = "16"
  instance_class = var.db_instance_class

  allocated_storage     = var.allocated_storage
  max_allocated_storage = var.allocated_storage * 2 # Auto-scaling hasta 2x
  storage_type          = "gp3"
  storage_encrypted     = true

  db_name  = var.db_name
  username = var.db_username
  password = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.rds.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  # ⚠️ PUBLICLY_ACCESSIBLE: Solo para desarrollo
  # En producción debe ser false para mayor seguridad
  # Si allow_public_access = true, también necesitas publicly_accessible = true
  publicly_accessible = var.allow_public_access # ⚠️ CAMBIAR A false EN PRODUCCIÓN

  backup_retention_period = 7
  backup_window           = "03:00-04:00"
  maintenance_window      = "mon:04:00-mon:05:00"

  skip_final_snapshot = true  # Para pruebas - cambiar a false en producción
  deletion_protection = false # Para pruebas - cambiar a true en producción
  # Nota: final_snapshot_identifier solo se usa cuando skip_final_snapshot = false
  # Si cambias skip_final_snapshot a false, descomenta la línea siguiente:
  # final_snapshot_identifier = "ewood-postgres-final-snapshot-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"

  # Performance Insights (opcional, puede aumentar costos)
  performance_insights_enabled = false

  # Monitoring
  monitoring_interval = 0 # 0 = deshabilitado, 60 = habilitado (requiere enhanced monitoring role)

  tags = {
    Name = "ewood-postgres-rds"
  }
}

