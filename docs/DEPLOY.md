# 🚀 Guía de Despliegue - Paso a Paso

## Resumen del Proceso

El despliegue se realiza en **2 etapas principales**:

1. **Infraestructura (Terraform)** - Crear recursos en AWS
2. **Aplicación (deploy.sh)** - Construir, subir imagen y desplegar en EC2

---

## 📋 Paso 1: Levantar la Infraestructura

### 1.1 Configurar Terraform

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
```

Editar `terraform.tfvars` con tus valores:

```hcl
aws_region          = "us-east-1"
ecr_repository_name = "ewood-backend"
s3_bucket_name      = "ewood-docker-compose-files"
instance_type       = "t3a.nano"
ami_id              = "ami-08c40ec9ead489470"
vpc_id              = "vpc-xxxxxxxxx"  # ⚠️ IMPORTANTE: Tu VPC ID
```

### 1.2 Crear Infraestructura

```bash
# Inicializar Terraform
terraform init

# Ver qué se va a crear (opcional pero recomendado)
terraform plan

# Crear la infraestructura
terraform apply
# Escribir "yes" cuando pregunte
```

### 1.3 Obtener IDs Necesarios

```bash
# Obtener el ID de la instancia EC2
terraform output -raw ec2_instance_id

# Obtener el nombre del bucket S3
terraform output -raw s3_bucket_name

# Obtener la IP pública (para probar después)
terraform output -raw ec2_public_ip

# Guardar la clave SSH (opcional)
terraform output -raw ssh_private_key > ../my-ec2-key.pem
chmod 400 ../my-ec2-key.pem
```

**Guarda estos valores** porque los necesitarás en el siguiente paso.

---

## 📋 Paso 2: Configurar Variables de Despliegue

### 2.1 Crear archivo `.env.push`

En la **raíz del proyecto** (no dentro de terraform):

```bash
cd ..  # Volver a la raíz
```

Crear `.env.push`:

```bash
export AWS_ACCOUNT_ID="262623998832"  # Tu Account ID de AWS
export AWS_REGION="us-east-1"
export ECR_REPOSITORY="ewood-backend"
export EC2_INSTANCE_ID="i-xxxxxxxxx"  # Del output de terraform
export S3_BUCKET_NAME="ewood-docker-compose-files"  # Del output de terraform
export IMAGE_TAG="latest"
```

**O mejor aún, usar los outputs de Terraform directamente:**

```bash
cd terraform
export AWS_ACCOUNT_ID="262623998832"
export AWS_REGION="us-east-1"
export ECR_REPOSITORY="ewood-backend"
export EC2_INSTANCE_ID=$(terraform output -raw ec2_instance_id)
export S3_BUCKET_NAME=$(terraform output -raw s3_bucket_name)
export IMAGE_TAG="latest"

# Guardar en archivo .env.push
cd ..
cat > .env.push << EOF
export AWS_ACCOUNT_ID="262623998832"
export AWS_REGION="us-east-1"
export ECR_REPOSITORY="ewood-backend"
export EC2_INSTANCE_ID="${EC2_INSTANCE_ID}"
export S3_BUCKET_NAME="${S3_BUCKET_NAME}"
export IMAGE_TAG="latest"
EOF
```

---

## 📋 Paso 3: Desplegar la Aplicación

### 3.1 Ejecutar el Script de Despliegue

```bash
# Cargar variables de entorno
source .env.push

# Ejecutar deploy
./deploy.sh
```

### 3.2 ¿Qué hace `deploy.sh`?

El script realiza automáticamente:

1. ✅ **Login en ECR** - Autenticación en AWS ECR
2. 🐳 **Construir imagen Docker** - Build de tu aplicación
3. 📤 **Push a ECR** - Subir imagen al repositorio
4. 📦 **Subir docker-compose.yml a S3** - Guardar configuración
5. 🚀 **Desplegar en EC2 vía SSM**:
   - Descargar docker-compose.yml desde S3
   - Login en ECR desde EC2
   - Hacer pull de la imagen
   - Detener contenedor anterior (si existe)
   - Iniciar nuevo contenedor

---

## 📋 Paso 4: Verificar Despliegue

### 4.1 Obtener IP Pública

```bash
cd terraform
EC2_IP=$(terraform output -raw ec2_public_ip)
echo $EC2_IP
```

### 4.2 Probar Conexión TCP

```bash
# Probar conexión al servidor TCP
nc -v $EC2_IP 8200
```

Si todo está bien, deberías poder conectarte y enviar mensajes.

### 4.3 Ver Logs del Despliegue (Opcional)

Si quieres ver los logs del proceso de despliegue en EC2:

```bash
# Listar comandos SSM recientes
aws ssm list-commands --instance-id $EC2_INSTANCE_ID --max-items 1

# Ver detalles de un comando específico
aws ssm get-command-invocation \
  --command-id <COMMAND_ID> \
  --instance-id $EC2_INSTANCE_ID
```

### 4.4 Ver Logs del Contenedor (Opcional)

```bash
# Conectarse a EC2 vía SSM
aws ssm start-session --target $EC2_INSTANCE_ID

# Dentro de la instancia, ver logs
docker logs ewood-golang-container

# Ver estado del contenedor
docker ps
```

---

## 🔄 Re-despliegue (Después de Cambios)

Cuando hagas cambios en el código:

```bash
# 1. Asegúrate de tener las variables cargadas
source .env.push

# 2. Ejecuta deploy de nuevo
./deploy.sh
```

El script automáticamente:
- Construirá la nueva imagen
- La subirá a ECR
- Detendrá el contenedor anterior
- Iniciará el nuevo contenedor

---

## 🧹 Destruir Todo (Cuando Termines)

### Eliminar Imágenes de ECR

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

### Destruir Infraestructura

```bash
cd terraform
terraform destroy
```

---

## 📝 Comandos Rápidos de Referencia

```bash
# === INFRAESTRUCTURA ===
cd terraform
terraform init
terraform plan
terraform apply
terraform output  # Ver todos los outputs
terraform destroy

# === DESPLIEGUE ===
source .env.push
./deploy.sh

# === VERIFICAR ===
EC2_IP=$(cd terraform && terraform output -raw ec2_public_ip)
nc -v $EC2_IP 8200

# === LOGS ===
aws ssm start-session --target $EC2_INSTANCE_ID
docker logs ewood-golang-container
```

---

## ⚠️ Notas Importantes

1. **Primera vez**: Debes levantar la infraestructura con Terraform antes de hacer deploy
2. **VPC ID**: Debes tener un VPC ID válido en `terraform.tfvars`
3. **Variables**: El archivo `.env.push` debe tener todos los valores correctos
4. **Permisos AWS**: Tu usuario/clave AWS debe tener permisos para:
   - ECR (crear repositorio, push imágenes)
   - EC2 (crear instancias, security groups)
   - S3 (crear bucket, subir archivos)
   - SSM (enviar comandos a instancias)
   - IAM (crear roles y políticas)
   - Terraform (crear/modificar recursos)

---

## 🐛 Troubleshooting

### Error: "ECR repository not found"
- Verifica que `terraform apply` se completó correctamente
- Verifica el nombre del repositorio en `.env.push`

### Error: "EC2 instance not found"
- Verifica que la instancia existe: `aws ec2 describe-instances --instance-ids $EC2_INSTANCE_ID`
- Asegúrate de usar el ID correcto del output de Terraform

### Error: "S3 bucket not found"
- Verifica que el bucket existe: `aws s3 ls | grep ewood`
- Verifica el nombre del bucket en `.env.push`

### El servidor no responde
- Verifica que el Security Group permite tráfico en el puerto 8200
- Verifica que el contenedor está corriendo: `docker ps` dentro de EC2
- Verifica los logs: `docker logs ewood-golang-container`

