# 🔍 Verificar Variables de Entorno

Guía para verificar que las variables de entorno estén correctamente configuradas tanto en local como en producción.

## 🏠 Verificar en Local

### Opción 1: Usando el endpoint `/env` (Recomendado)

```bash
# Ver variables de entorno formateadas
make check-env

# O directamente con curl
curl http://localhost:8200/env | jq
```

### Opción 2: Ver directamente del contenedor

```bash
# Ver variables sin contraseñas
make check-env-container

# O directamente
docker exec ewood-challenge env | grep -E "POSTGRES|MONGODB|ENV|PORT" | grep -v PASSWORD
```

### Opción 3: Ver variables sin formato JSON

```bash
make check-env-raw
```

## 🚀 Verificar en Producción

### Opción 1: Usando el endpoint `/env` (Recomendado)

```bash
# Obtener IP de EC2
cd terraform
EC2_IP=$(terraform output -raw ec2_public_ip)

# Ver variables de entorno
curl http://$EC2_IP:8200/env | jq
```

### Opción 2: Ver directamente del contenedor en EC2

```bash
# Ver variables sin contraseñas vía SSM
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["docker exec ewood-golang-container env | grep -E \"POSTGRES|MONGODB|ENV|PORT\" | grep -v PASSWORD"]'
```

### Opción 3: Ver el archivo env.production en EC2

```bash
# Ver el archivo (sin mostrar contraseñas)
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["cat /home/ubuntu/env.production | grep -v PASSWORD"]'
```

## 📋 Respuesta Esperada del Endpoint `/env`

### Local (con Docker Compose)

```json
{
  "timestamp": "2025-11-30T...",
  "environment": {
    "ENV": "local",
    "PORT": ":8080"
  },
  "postgresql": {
    "POSTGRES_HOST": "postgres",
    "POSTGRES_PORT": "5432",
    "POSTGRES_USER": "***",
    "POSTGRES_PASSWORD": "***",
    "POSTGRES_DB": "ewood_db",
    "POSTGRES_SSLMODE": "disable",
    "configured": true
  },
  "mongodb": {
    "MONGODB_URI": "mongodb://***:***@mongodb:27017/ewood?authSource=admin",
    "MONGODB_DATABASE": "ewood",
    "configured": true
  }
}
```

### Producción (con RDS)

```json
{
  "timestamp": "2025-11-30T...",
  "environment": {
    "ENV": "production",
    "PORT": ":8080"
  },
  "postgresql": {
    "POSTGRES_HOST": "ewood-postgres.xxxxx.us-east-1.rds.amazonaws.com",
    "POSTGRES_PORT": "5432",
    "POSTGRES_USER": "***",
    "POSTGRES_PASSWORD": "***",
    "POSTGRES_DB": "ewood_db",
    "POSTGRES_SSLMODE": "require",
    "configured": true
  },
  "mongodb": {
    "MONGODB_URI": "<not_set>",
    "MONGODB_DATABASE": "",
    "configured": false
  }
}
```

## ✅ Checklist de Verificación

### Variables de PostgreSQL (Requeridas)

- [ ] `POSTGRES_HOST` está configurado
- [ ] `POSTGRES_PORT` está configurado (o usa default 5432)
- [ ] `POSTGRES_USER` está configurado
- [ ] `POSTGRES_PASSWORD` está configurado (mostrado como `***`)
- [ ] `POSTGRES_DB` está configurado
- [ ] `POSTGRES_SSLMODE` es apropiado:
  - Local: `disable`
  - Producción: `require` o `prefer`

### Variables de MongoDB (Opcionales)

- [ ] `MONGODB_URI` está configurado (si usas MongoDB)
- [ ] `MONGODB_DATABASE` está configurado (si usas MongoDB)

### Variables del Servidor

- [ ] `ENV` está configurado (`local` o `production`)
- [ ] `PORT` está configurado (default: `:8080`)

## 🔧 Comandos Útiles

### Ver todas las variables (incluyendo contraseñas) - Solo para debug local

```bash
# Local
docker exec ewood-challenge env | grep -E "POSTGRES|MONGODB"

# Producción (vía SSM)
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["docker exec ewood-golang-container env | grep POSTGRES"]'
```

### Probar conexión a PostgreSQL

```bash
# Verificar que PostgreSQL esté conectado
curl http://localhost:8200/health | jq

# Debería mostrar:
# {
#   "status": "ok",
#   "services": {
#     "postgresql": "connected"
#   }
# }
```

## 🐛 Troubleshooting

### `"configured": false` en PostgreSQL

**Causa**: `POSTGRES_HOST` no está configurado.

**Solución**: 
- Local: Verifica `docker-compose-local.yml`
- Producción: Verifica que `env.production` esté en EC2 y se haya descargado desde S3

### `"POSTGRES_HOST": ""` (vacío)

**Causa**: La variable no está siendo cargada correctamente.

**Solución**: 
- Verifica que el archivo `env.production` esté en `/home/ubuntu/env.production` en EC2
- Verifica que `docker-compose.yml` tenga `env_file` configurado

### Error de conexión en `/health`

**Causa**: Las credenciales son incorrectas o el host no es accesible.

**Solución**:
- Verifica las credenciales del RDS
- Verifica que el Security Group de RDS permita conexiones desde EC2
- Verifica que `POSTGRES_SSLMODE` sea correcto para tu entorno

## 📚 Endpoints Disponibles

- `/check` - Healthcheck básico del servidor
- `/health` - Estado de conexiones a bases de datos
- `/env` - Variables de entorno configuradas (sin contraseñas)

