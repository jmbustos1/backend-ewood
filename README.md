# 🚀 Ewood Backend - Sistema de Infraestructura y Despliegue

Proyecto backend basado en Go con infraestructura completa en AWS, incluyendo EC2, ECR, S3, IAM, Security Groups y SSM para despliegue automatizado.

---

## 📋 Tabla de Contenidos

- [Arquitectura](#arquitectura)
- [Requisitos](#requisitos)
- [Infraestructura](#infraestructura)
- [Despliegue](#despliegue)
- [Desarrollo Local](#desarrollo-local)
- [Servidor TCP](#servidor-tcp)
- [Estructura del Proyecto](#estructura-del-proyecto)

---

## 🏗️ Arquitectura

### Componentes de Infraestructura

- **EC2**: Instancia para ejecutar el servidor TCP/HTTP
- **ECR**: Repositorio Docker para almacenar imágenes
- **S3**: Bucket para almacenar archivos de configuración (docker-compose.yml)
- **IAM**: Roles y políticas para acceso a servicios AWS
- **Security Group**: Reglas de firewall para EC2
- **SSM**: Sistema de gestión de instancias sin necesidad de SSH

### Flujo de Despliegue

1. Construcción de imagen Docker
2. Push a ECR
3. Upload de docker-compose.yml a S3
4. Despliegue en EC2 vía SSM
5. Pull de imagen y ejecución del contenedor

---

## ⚙️ Requisitos

### Herramientas necesarias

- **AWS CLI** configurado con credenciales
- **Terraform** >= 1.0
- **Docker** y Docker Compose
- **Go** >= 1.22 (para desarrollo local)
- **netcat** (nc) para probar conexiones TCP

### Variables de entorno

Crea un archivo `.env.push` con las siguientes variables:

```bash
export AWS_ACCOUNT_ID="<tu_account_id>"
export AWS_REGION="us-east-1"
export ECR_REPOSITORY="ewood-backend"
export EC2_INSTANCE_ID="<instance_id>"
export S3_BUCKET_NAME="ewood-docker-compose-files"
```

---

## 🧱 Despliegue de Infraestructura

### 📝 Configuración Inicial

1. **Editar variables de Terraform**:

   Crea `terraform/terraform.tfvars`:

   ```hcl
   aws_region          = "us-east-1"
   ecr_repository_name = "ewood-backend"
   s3_bucket_name      = "ewood-docker-compose-files"
   instance_type       = "t3a.nano"
   ami_id              = "ami-08c40ec9ead489470"  # Ubuntu 22.04
   vpc_id              = "vpc-xxxxxxxxx"  # Tu VPC ID
   ```

2. **Inicializar Terraform**:

   ```bash
   cd terraform
   terraform init
   ```

### ✅ Levantar Infraestructura

```bash
cd terraform
terraform apply
```

Obtener el ID de instancia EC2:

```bash
aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=ec2-ewood" \
  --query "Reservations[*].Instances[*].InstanceId" \
  --output text
```

### 🔑 Obtener Clave SSH

```bash
terraform output -raw ssh_private_key > ../my-ec2-key.pem
chmod 400 my-ec2-key.pem
```

### 🖥️ Conexión SSH (Opcional)

```bash
ssh -i my-ec2-key.pem ubuntu@<EC2-IP>
```

**Nota**: Se recomienda usar SSM en lugar de SSH para mayor seguridad:

```bash
aws ssm start-session --target <INSTANCE_ID>
```

### 🧨 Destruir Infraestructura

**Importante**: Antes de destruir, eliminar imágenes de ECR:

```bash
aws ecr list-images \
  --repository-name ewood-backend \
  --query 'imageIds[*]' \
  --output json | \
  jq -c '.[]' | \
  while read img; do
    aws ecr batch-delete-image \
      --repository-name ewood-backend \
      --image-ids "$img"
  done
```

Luego destruir la infraestructura:

```bash
cd terraform
terraform destroy
```

---

## 📦 Despliegue de la Aplicación

### Preparar Variables

```bash
# Desde el directorio raíz del proyecto
source .env.push
export S3_BUCKET_NAME=$(cd terraform && terraform output -raw s3_bucket_name)
export EC2_INSTANCE_ID=$(cd terraform && terraform output -raw ec2_instance_id)
```

### Ejecutar Despliegue

```bash
chmod +x deploy.sh
./deploy.sh
```

El script `deploy.sh` realiza:

1. ✅ Login en ECR
2. 🐳 Construcción de imagen Docker
3. 📤 Push a ECR
4. 📦 Upload de docker-compose.yml a S3
5. 🚀 Despliegue en EC2 vía SSM

### Verificar Despliegue

```bash
# Obtener IP pública de EC2
cd terraform
terraform output ec2_public_ip

# Probar conexión TCP
nc -v <EC2-IP> 8200
```

### Ver Logs de EC2

```bash
# Ver logs de cloud-init
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"<INSTANCE_ID>\"]}]" \
  --parameters "commands=[\"sudo cat /var/log/cloud-init-output.log\"]" \
  --query "Command.CommandId" \
  --output text

# Ver logs del contenedor
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"<INSTANCE_ID>\"]}]" \
  --parameters "commands=[\"docker logs ewood-golang-container\"]"
```

---

## 💻 Desarrollo Local

### Ejecutar Servidor TCP Localmente

```bash
# Opción 1: Ejecutar directamente con Go
go run tcp_server.go

# Opción 2: Ejecutar con Docker Compose
make dockerup

# Opción 3: Compilar y ejecutar
go build -o server tcp_server.go
./server
```

### Probar Conexión TCP Local

En una terminal separada:

```bash
nc localhost 8200
```

O usar el comando Make:

```bash
make test-tcp
```

### Detener Servicios Locales

```bash
make dockerdown
```

---

## 🌐 Servidor TCP

### Descripción

El servidor TCP (`tcp_server.go`) es un servidor simple que:

- ✅ Escucha en el puerto 8080 (mapeado a 8200 externamente)
- ✅ Acepta múltiples conexiones concurrentes
- ✅ Responde con eco de mensajes
- ✅ Incluye mensajes de bienvenida y timestamps
- ✅ Permite desconexión con comando `QUIT`

### Comandos del Cliente

- Cualquier mensaje: El servidor responde con un eco
- `QUIT`: Cierra la conexión

### Ejemplo de Uso

```bash
# Conectar al servidor
nc localhost 8200

# El servidor responderá:
# ¡Bienvenido al servidor TCP de Ewood!
# Conexión desde: 127.0.0.1:xxxxx
# Hora del servidor: 2025-01-XX...
# Envía 'QUIT' para desconectarte.

# Enviar mensaje
Hola servidor

# Respuesta:
# [15:30:45] Echo: Hola servidor

# Desconectar
QUIT
```

---

## 📁 Estructura del Proyecto

Este proyecto sigue una **Clean Architecture** para mantener el código organizado y fácil de mantener.

```
backend-ewood/
├── cmd/
│   └── server/
│       └── main.go              # Punto de entrada de la aplicación
├── internal/                    # Código interno del proyecto
│   ├── handler/                 # Handlers HTTP
│   │   ├── check.go            # Handler para /check
│   │   ├── health.go           # Handler para /health
│   │   └── env.go              # Handler para /env
│   ├── router/                  # Configuración de rutas
│   │   └── router.go           # Definición de rutas HTTP
│   ├── service/                 # Lógica de negocio (futuro)
│   └── client/                  # Clientes de bases de datos
│       ├── postgres/           # Cliente de PostgreSQL
│       └── mongodb/            # Cliente de MongoDB (futuro)
├── config/                      # Configuración de la aplicación
│   └── config.go               # Carga de variables de entorno
├── docs/                        # Documentación del proyecto
│   ├── ESTRUCTURA_PROYECTO.md  # Documentación de la arquitectura
│   ├── DEPLOY.md               # Guía de despliegue
│   └── ...                     # Otra documentación
├── terraform/                   # Infraestructura como código
│   ├── main.tf                # Configuración principal
│   ├── variables.tf           # Variables de Terraform
│   ├── outputs.tf             # Outputs de Terraform
│   └── modules/               # Módulos de Terraform
│       ├── ec2/               # Módulo EC2
│       ├── ecr/               # Módulo ECR
│       ├── iam/               # Módulo IAM
│       ├── rds/               # Módulo RDS
│       ├── s3/                # Módulo S3
│       ├── security-group/    # Módulo Security Group
│       └── ssm/               # Módulo SSM
├── Dockerfile                  # Imagen Docker
├── docker-compose.yml          # Docker Compose para producción
├── docker-compose-local.yml    # Docker Compose para desarrollo
├── deploy.sh                   # Script de despliegue
├── Makefile                    # Comandos útiles
├── go.mod                      # Dependencias Go
└── README.md                   # Este archivo
```

Para más detalles sobre la arquitectura, consulta [docs/ESTRUCTURA_PROYECTO.md](docs/ESTRUCTURA_PROYECTO.md).

---

## 🔧 Comandos Útiles (Makefile)

```bash
# Desarrollo
make dockerup           # Levantar contenedor local
make dockerdown         # Detener contenedor local
make build-no-cache     # Reconstruir sin cache
make test-tcp           # Probar conexión TCP
make run-local          # Ejecutar servidor localmente
make test               # Ejecutar tests
```

---

## 🔐 Seguridad

### Mejores Prácticas Implementadas

- ✅ Security Groups con reglas específicas
- ✅ S3 bucket con encriptación y acceso bloqueado
- ✅ IAM roles con permisos mínimos necesarios
- ✅ SSM para gestión sin necesidad de SSH público
- ✅ ECR con escaneo de imágenes habilitado

### Recomendaciones Adicionales

- 🔒 Restringir acceso SSH por IP específica en Security Group
- 🔒 Usar AWS Systems Manager Session Manager en lugar de SSH
- 🔒 Rotar claves SSH regularmente
- 🔒 Habilitar CloudTrail para auditoría
- 🔒 Usar Secrets Manager para credenciales sensibles

---

## 📝 Notas

- El puerto 8200 está mapeado al 8080 del contenedor
- El docker-compose.yml se descarga desde S3 durante el despliegue
- El servidor TCP acepta múltiples conexiones concurrentes

---

## 🐛 Troubleshooting

### Error: "Cannot connect to ECR"

```bash
# Verificar autenticación
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com
```

### Error: "EC2 instance not found"

Verificar que la instancia existe y el ID es correcto:

```bash
aws ec2 describe-instances --instance-ids <INSTANCE_ID>
```

### Error: "SSM agent not running"

```bash
# Conectarse a la instancia y verificar
aws ssm start-session --target <INSTANCE_ID>
sudo systemctl status amazon-ssm-agent
```

### Ver logs de despliegue SSM

```bash
# Obtener Command ID del último despliegue
aws ssm list-commands --instance-id <INSTANCE_ID> --max-items 1

# Ver detalles
aws ssm get-command-invocation \
  --command-id <COMMAND_ID> \
  --instance-id <INSTANCE_ID>
```

---

## 📚 Referencias

- [AWS ECR Documentation](https://docs.aws.amazon.com/ecr/)
- [AWS SSM Documentation](https://docs.aws.amazon.com/systems-manager/)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [Go net package](https://pkg.go.dev/net)

---

## 📄 Licencia

Este proyecto está basado en la estructura del proyecto desafio-plutto.
