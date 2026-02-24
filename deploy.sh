#!/bin/bash
set -euo pipefail

# ================================
# 1. CONFIGURACIÓN
# ================================
AWS_REGION="${AWS_REGION:-us-east-1}"
AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:?Variable AWS_ACCOUNT_ID no definida}"
ECR_REPOSITORY="${ECR_REPOSITORY:-ewood-backend}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
EC2_INSTANCE_ID="${EC2_INSTANCE_ID:?Variable EC2_INSTANCE_ID no definida}"
S3_BUCKET_NAME="${S3_BUCKET_NAME:?Variable S3_BUCKET_NAME no definida}"

IMAGE_URI="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_REPOSITORY:$IMAGE_TAG"

echo "🌍 AWS_REGION=$AWS_REGION"
echo "🧾 AWS_ACCOUNT_ID=$AWS_ACCOUNT_ID"
echo "📦 ECR_REPOSITORY=$ECR_REPOSITORY"
echo "🏷 IMAGE_TAG=$IMAGE_TAG"
echo "🖥 EC2_INSTANCE_ID=$EC2_INSTANCE_ID"
echo "🪣 S3_BUCKET_NAME=$S3_BUCKET_NAME"
echo "🐳 IMAGE_URI=$IMAGE_URI"

# ================================
# 2. LOGIN CORRECTO A ECR (LOCAL)
# ================================
echo "🔑 Login local en ECR..."
aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

# ================================
# 3. BUILD + PUSH
# ================================
echo "🐳 Construyendo imagen Docker..."
docker build -t "$ECR_REPOSITORY" .

echo "🏷 Etiquetando imagen..."
docker tag "$ECR_REPOSITORY:$IMAGE_TAG" "$IMAGE_URI"

echo "📤 Subiendo imagen a ECR..."
docker push "$IMAGE_URI"

# ================================
# 4. SUBIR DOCKER-COMPOSE A S3
# ================================
echo "📦 Subiendo docker-compose.yml a S3..."
aws s3 cp docker-compose.yml "s3://$S3_BUCKET_NAME/docker-compose.yml" --region "$AWS_REGION"

# ================================
# 4.5 SUBIR ENV.PRODUCTION A S3 (si existe)
# ================================
if [ -f env.production ]; then
    echo "📦 Subiendo env.production a S3..."
    aws s3 cp env.production "s3://$S3_BUCKET_NAME/env.production" --region "$AWS_REGION"
else
    echo "⚠️  Advertencia: env.production no existe, se usará configuración por defecto"
fi

# ================================
# 5. ENVIAR COMANDO SSM (DEPLOY)
# ================================
echo "🚀 Enviando comando SSM para desplegar en EC2..."

COMMAND_ID=$(aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters "commands=[
    \"echo '[DEPLOY] Iniciando...' > /tmp/deploy.log\",
    \"echo '[DEPLOY] Descargando docker-compose.yml desde S3...' >> /tmp/deploy.log\",
    \"aws s3 cp s3://$S3_BUCKET_NAME/docker-compose.yml /home/ubuntu/docker-compose.yml --region $AWS_REGION >> /tmp/deploy.log 2>&1\",
    \"echo '[DEPLOY] Descargando env.production desde S3...' >> /tmp/deploy.log\",
    \"aws s3 cp s3://$S3_BUCKET_NAME/env.production /home/ubuntu/env.production --region $AWS_REGION >> /tmp/deploy.log 2>&1 || echo '[DEPLOY] Advertencia: env.production no encontrado en S3' >> /tmp/deploy.log\",
    \"echo '[DEPLOY] Login en ECR...' >> /tmp/deploy.log\",
    \"docker login --username AWS --password \$(aws ecr get-login-password --region $AWS_REGION) $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com >> /tmp/deploy.log 2>&1\",
    \"echo '[DEPLOY] Pull imagen...' >> /tmp/deploy.log\",
    \"docker pull $IMAGE_URI >> /tmp/deploy.log 2>&1\",
    \"echo '[DEPLOY] docker compose down...' >> /tmp/deploy.log\",
    \"docker compose -f /home/ubuntu/docker-compose.yml down >> /tmp/deploy.log 2>&1 || true\",
    \"echo '[DEPLOY] docker compose up...' >> /tmp/deploy.log\",
    \"docker compose -f /home/ubuntu/docker-compose.yml up -d >> /tmp/deploy.log 2>&1\",
    \"echo '[DEPLOY] Esperando que el contenedor esté listo...' >> /tmp/deploy.log\",
    \"sleep 5\",
    \"echo '[DEPLOY] Ejecutando migraciones...' >> /tmp/deploy.log\",
    \"docker exec ewood-golang-container /app/migrate -command=up -path=/app/migrations >> /tmp/deploy.log 2>&1 || echo '[DEPLOY] Advertencia: Error ejecutando migraciones (puede ser que no haya migraciones pendientes)' >> /tmp/deploy.log\",
    \"echo '[DEPLOY] Finalizado con éxito.' >> /tmp/deploy.log\"
  ]" \
  --query "Command.CommandId" \
  --output text)

echo "✅ Despliegue iniciado con Command ID: $COMMAND_ID"

echo ""
echo "📄 Ver logs del deploy:"
echo "   aws ssm get-command-invocation --command-id $COMMAND_ID --instance-id $EC2_INSTANCE_ID --query 'StandardOutputContent' --output text"
echo ""
echo "📄 Logs completos en EC2:"
echo "   aws ssm send-command --document-name AWS-RunShellScript --targets '[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]' --parameters 'commands=[\"cat /tmp/deploy.log\"]'"
echo ""
echo "🟢 Done!"
