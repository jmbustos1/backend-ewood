# 🔐 Gestión de Secrets y Variables de Entorno

## 📋 Mejores Prácticas

### ✅ Lo Correcto (Producción)

Tienes razón en tu intuición. La mejor práctica es:

#### 1. **Variables de Operación (Runtime Secrets) → AWS Secrets Manager**
- Contraseñas de bases de datos
- API keys de servicios externos
- Tokens de autenticación
- Certificados privados
- Cualquier información sensible que la aplicación necesita en runtime

**Ejemplos:**
- `POSTGRES_PASSWORD`
- `MONGODB_URI` (con credenciales)
- `JWT_SECRET`
- `AWS_ACCESS_KEY_ID` (si es necesario)
- `STRIPE_SECRET_KEY`

#### 2. **Variables de Despliegue (CI/CD Config) → GitHub Actions Secrets**
- IDs de recursos AWS (EC2, ECR, S3)
- Nombres de buckets/repositorios
- Tags de imágenes
- Configuración de CI/CD
- Variables no sensibles de configuración

**Ejemplos:**
- `AWS_ACCOUNT_ID`
- `EC2_INSTANCE_ID`
- `S3_BUCKET_NAME`
- `ECR_REPOSITORY`
- `AWS_REGION`
- `IMAGE_TAG`

---

## 🏗️ Arquitectura Recomendada

### Estado Actual (Desarrollo)
```
┌─────────────────────────────────────┐
│  env.production (archivo local)     │
│  - Subido a S3                      │
│  - Descargado en EC2                │
│  - Cargado por docker-compose        │
└─────────────────────────────────────┘
```

### Estado Ideal (Producción)
```
┌─────────────────────────────────────┐
│  GitHub Actions Secrets              │
│  - AWS_ACCOUNT_ID                    │
│  - EC2_INSTANCE_ID                   │
│  - S3_BUCKET_NAME                    │
│  - ECR_REPOSITORY                    │
└─────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  AWS Secrets Manager                 │
│  - POSTGRES_PASSWORD                 │
│  - MONGODB_URI                       │
│  - JWT_SECRET                        │
│  - API_KEYS                          │
└─────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│  EC2 Instance (Runtime)              │
│  - Lee secrets desde Secrets Manager │
│  - Inyecta en contenedor             │
└─────────────────────────────────────┘
```

---

## 🔄 Migración Recomendada

### Paso 1: Mover Secrets a AWS Secrets Manager

```bash
# Crear secret para PostgreSQL
aws secretsmanager create-secret \
  --name ewood/postgres/production \
  --secret-string '{
    "POSTGRES_HOST": "ewood-postgres.c3u0i2kuisp0.us-east-1.rds.amazonaws.com",
    "POSTGRES_PORT": "5432",
    "POSTGRES_USER": "ewood_admin",
    "POSTGRES_PASSWORD": "Jif)-DbP>p}MoEe3",
    "POSTGRES_DB": "ewood_db",
    "POSTGRES_SSLMODE": "require"
  }'

# Crear secret para MongoDB (cuando lo implementes)
aws secretsmanager create-secret \
  --name ewood/mongodb/production \
  --secret-string '{
    "MONGODB_URI": "mongodb://...",
    "MONGODB_DATABASE": "ewood"
  }'
```

### Paso 2: Configurar IAM para Acceso a Secrets

El rol IAM de EC2 debe tener permiso para leer secrets:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:us-east-1:*:secret:ewood/*"
    }
  ]
}
```

### Paso 3: Actualizar docker-compose.yml

```yaml
services:
  golang-tcp-server:
    image: ${ECR_IMAGE}
    container_name: ewood-golang-container
    ports:
      - "8200:8080"
    restart: unless-stopped
    # Leer secrets desde Secrets Manager (vía script)
    environment:
      - ENV=production
      - POSTGRES_HOST=${POSTGRES_HOST}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      # ... etc
```

### Paso 4: Script para Cargar Secrets en EC2

```bash
#!/bin/bash
# /home/ubuntu/load-secrets.sh

# Obtener secrets desde Secrets Manager
SECRETS=$(aws secretsmanager get-secret-value \
  --secret-id ewood/postgres/production \
  --query SecretString \
  --output text)

# Parsear JSON y exportar variables
export POSTGRES_HOST=$(echo $SECRETS | jq -r '.POSTGRES_HOST')
export POSTGRES_PASSWORD=$(echo $SECRETS | jq -r '.POSTGRES_PASSWORD')
# ... etc

# Ejecutar docker-compose con variables
docker compose -f /home/ubuntu/docker-compose.yml up -d
```

### Paso 5: Configurar GitHub Actions Secrets

En GitHub → Settings → Secrets and variables → Actions:

- `AWS_ACCOUNT_ID`
- `AWS_REGION`
- `EC2_INSTANCE_ID`
- `S3_BUCKET_NAME`
- `ECR_REPOSITORY`
- `AWS_ACCESS_KEY_ID` (para CI/CD)
- `AWS_SECRET_ACCESS_KEY` (para CI/CD)

---

## 📊 Comparación: Actual vs Recomendado

| Aspecto | Actual (S3) | Recomendado (Secrets Manager) |
|---------|-------------|-------------------------------|
| **Seguridad** | ⚠️ Archivo plano en S3 | ✅ Encriptado, rotación automática |
| **Auditoría** | ⚠️ Limitada | ✅ CloudTrail completo |
| **Rotación** | ❌ Manual | ✅ Automática |
| **Versionado** | ⚠️ Manual (S3 versioning) | ✅ Automático |
| **Acceso** | ⚠️ IAM policies | ✅ IAM + Resource policies |
| **Costo** | ✅ Muy bajo | ⚠️ ~$0.40/secret/mes |

---

## 🎯 Resumen

### ✅ Para Producción:

1. **GitHub Actions Secrets** → Variables de despliegue (no sensibles)
2. **AWS Secrets Manager** → Secrets de runtime (sensibles)
3. **EC2 IAM Role** → Permisos para leer secrets
4. **Script de carga** → Lee secrets y los inyecta en contenedor

### ⚠️ Estado Actual (OK para Desarrollo):

- `env.production` en S3 funciona, pero no es ideal para producción
- No hay rotación automática
- Menos auditoría
- Más difícil de gestionar múltiples ambientes

---

## 🚀 Próximos Pasos

1. Crear secrets en AWS Secrets Manager
2. Actualizar IAM role de EC2
3. Crear script de carga de secrets
4. Actualizar docker-compose.yml
5. Configurar GitHub Actions Secrets
6. Migrar deploy.sh a GitHub Actions workflow

