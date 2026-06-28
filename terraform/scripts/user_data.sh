#!/bin/bash

# Actualizar paquetes
apt update -y
apt upgrade -y

apt install -y ca-certificates curl gnupg lsb-release

mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Activa Docker y añade el usuario
systemctl enable docker
systemctl start docker
usermod -aG docker ubuntu

# Instalar AWS CLI v2
cd /tmp
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
apt install -y unzip
unzip awscliv2.zip
./aws/install

# 🔹 Eliminar el agente SSM si ya está instalado con Snap (para evitar conflicto con dpkg)
if snap list | grep -q amazon-ssm-agent; then
    snap remove amazon-ssm-agent
fi

# 🔹 Instalar el agente AWS SSM con dpkg
cd /tmp
curl -O https://s3.amazonaws.com/ec2-downloads-windows/SSMAgent/latest/debian_amd64/amazon-ssm-agent.deb
dpkg -i amazon-ssm-agent.deb

# 🔹 Habilitar y arrancar el servicio SSM
systemctl enable amazon-ssm-agent
systemctl start amazon-ssm-agent

# 🔹 Verificar que el servicio está corriendo
systemctl status amazon-ssm-agent --no-pager

# ─── Caddy reverse proxy + HTTPS (Let's Encrypt) ────────────────────────────
# Termina TLS en 443 hacia el exterior y reenvia al backend Go que escucha
# solo en localhost:8200 (no expuesto en el security group).
# Caddy gestiona el certificado automaticamente via HTTP-01 challenge.
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/gpg.key | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt | tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt install -y caddy

# Caddyfile inicial. El backend Go se conecta en localhost:8200.
# El dominio api.exwood.dev tiene que estar ya apuntando al EIP de esta EC2
# para que Let's Encrypt resuelva el HTTP-01 challenge.
cat > /etc/caddy/Caddyfile <<'CADDYFILE'
api.exwood.dev {
    reverse_proxy localhost:8200
}
CADDYFILE

systemctl reload caddy
