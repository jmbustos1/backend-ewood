resource "aws_security_group" "tcp_sg" {
  name        = "ec2-ewood-sg"
  description = "Security Group para el servidor TCP en EC2"
  vpc_id      = var.vpc_id

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }

  # Permitir tráfico HTTP/WebSocket en el puerto 8200 (mapeado al 8080 interno)
  # Protocolo TCP es correcto - HTTP y WebSockets corren sobre TCP
  ingress {
    from_port   = 8200
    to_port     = 8200
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP API and WebSocket for scooters"
  }

  # Permitir tráfico TCP en el puerto 8201 para recibir actualizaciones de ubicación de scooters
  ingress {
    from_port   = 8201
    to_port     = 8201
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "TCP for scooter location updates"
  }

  # Permitir salida de tráfico para que EC2 pueda hacer pull de ECR
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Permitir acceso a SSM (puerto 443 para comunicación con AWS)
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Permitir tráfico en el puerto 22 (SSH)
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # ⚠️ Cambia esto por TU IP para seguridad
  }

  tags = {
    Name = "ewood-tcp-server-sg"
  }
}
