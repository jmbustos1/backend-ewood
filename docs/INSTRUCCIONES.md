# 📋 Instrucciones Rápidas de Despliegue

## 🚀 Inicio Rápido

### 1. Configurar Variables de Terraform

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Editar terraform.tfvars con tus valores (especialmente vpc_id)
```

### 2. Levantar Infraestructura

```bash
terraform init
terraform plan  # Revisar cambios
terraform apply
```

Obtener outputs importantes:

```bash
terraform output ec2_public_ip
terraform output ec2_instance_id
terraform output s3_bucket_name
terraform output ssh_private_key > ../my-ec2-key.pem
chmod 400 ../my-ec2-key.pem
```

### 3. Configurar Variables de Despliegue

Crear `.env.push` en la raíz del proyecto:

```bash
export AWS_ACCOUNT_ID="<tu_account_id>"
export AWS_REGION="us-east-1"
export ECR_REPOSITORY="ewood-backend"
export EC2_INSTANCE_ID="<obtener con: terraform output -raw ec2_instance_id>"
export S3_BUCKET_NAME="<obtener con: terraform output -raw s3_bucket_name>"
```

### 4. Desplegar Aplicación

```bash
source .env.push
./deploy.sh
```

### 5. Probar Conexión

```bash
# Obtener IP pública
cd terraform
EC2_IP=$(terraform output -raw ec2_public_ip)

# Probar conexión TCP
nc -v $EC2_IP 8200
```

---

## 🧹 Limpieza

### Destruir Infraestructura

```bash
# 1. Eliminar imágenes de ECR
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

# 2. Destruir infraestructura
cd terraform
terraform destroy
```

---

## 🔍 Comandos Útiles

### Ver Logs de EC2

```bash
# Cloud-init
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"<INSTANCE_ID>\"]}]" \
  --parameters "commands=[\"sudo cat /var/log/cloud-init-output.log\"]"

# Logs del contenedor
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"<INSTANCE_ID>\"]}]" \
  --parameters "commands=[\"docker logs ewood-golang-container\"]"
```

### Conectar a EC2 vía SSM (sin SSH)

```bash
aws ssm start-session --target <INSTANCE_ID>
```

### Ver Estado del Contenedor

```bash
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"<INSTANCE_ID>\"]}]" \
  --parameters "commands=[\"docker ps -a\"]"
```

---

## 📝 Notas Importantes

- El puerto 8200 está mapeado al 8080 del contenedor
- El servidor TCP acepta múltiples conexiones concurrentes
- El docker-compose.yml se almacena y descarga desde S3

---

## 🐛 Problemas Comunes

### Error: "ECR repository not found"

Verificar que el repositorio existe:

```bash
aws ecr describe-repositories --repository-names ewood-backend
```

Si no existe, ejecutar `terraform apply` de nuevo.

### Error: "S3 bucket not found"

Verificar que el bucket existe:

```bash
aws s3 ls | grep ewood
```

### Error: "SSM command failed"

Verificar logs del comando:

```bash
aws ssm list-commands --instance-id <INSTANCE_ID> --max-items 1
aws ssm get-command-invocation --command-id <COMMAND_ID> --instance-id <INSTANCE_ID>
```

### El servidor no responde

1. Verificar que el Security Group permite tráfico en el puerto 8200
2. Verificar que el contenedor está corriendo:
   ```bash
   aws ssm send-command \
     --document-name "AWS-RunShellScript" \
     --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"<INSTANCE_ID>\"]}]" \
     --parameters "commands=[\"docker ps\"]"
   ```
3. Verificar logs del contenedor (comando arriba)
