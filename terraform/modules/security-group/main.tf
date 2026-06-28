resource "aws_security_group" "tcp_sg" {
  name        = "ec2-ewood-sg"
  description = "Security Group para el servidor Ewood (HTTPS via Caddy + TCP IoT)"
  vpc_id      = var.vpc_id

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [description]
  }

  # HTTPS publico — terminado por Caddy en EC2
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS publico (Caddy reverse proxy al backend)"
  }

  # HTTP publico — Caddy lo usa para HTTP-01 challenge de Let's Encrypt
  # y para redirigir 80 -> 443 automaticamente
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP publico (Let's Encrypt HTTP-01 + redirect a HTTPS)"
  }

  # TCP plano para telemetria de scooters ESP32+SIM7600
  # (sin TLS — autenticacion via DEVICE_SECRET en el primer mensaje)
  ingress {
    from_port   = 8201
    to_port     = 8201
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "TCP para telemetria de scooters IoT"
  }

  # SSH
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # TODO: limitar a IP del operador cuando este definida
    description = "SSH para administracion"
  }

  # NOTA: el puerto 8200 (backend Go) NO se expone publicamente.
  # Solo Caddy en localhost lo consume y termina TLS hacia el exterior en 443.

  # Egress libre (pull ECR, comunicacion con AWS, Let's Encrypt, MercadoPago, etc.)
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "ewood-backend-sg"
  }
}
