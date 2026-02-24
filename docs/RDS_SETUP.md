# 🗄️ Configuración de RDS PostgreSQL

## ✅ Lo que se ha implementado

### 1. Módulo Terraform para RDS
- ✅ `terraform/modules/rds/` - Módulo completo para RDS PostgreSQL
- ✅ Security Group configurado para permitir conexiones desde EC2
- ✅ Subnet Group automático (usa subnets de la VPC)
- ✅ Instancia pequeña `db.t3.micro` para pruebas
- ✅ Contraseña generada automáticamente si no se proporciona

### 2. Código Go Idempotente
- ✅ `database/postgres.go` - Conexión a PostgreSQL
- ✅ Funciona tanto en local como en producción
- ✅ Lee variables de entorno automáticamente
- ✅ Endpoint `/health` para verificar conexión

### 3. Integración
- ✅ `tcp_server.go` actualizado para conectar a PostgreSQL
- ✅ Cierre graceful de conexiones
- ✅ Manejo de errores robusto

## 🚀 Pasos para Levantar RDS

### Paso 1: Actualizar terraform.tfvars

Agrega estas variables a tu `terraform/terraform.tfvars`:

```hcl
# Variables existentes...
vpc_id = "vpc-xxxxxxxxx"

# Nuevas variables para RDS (opcionales)
db_username = "ewood_admin"           # Opcional, default: ewood_admin
db_password = ""                      # Opcional, se genera automáticamente si está vacío
db_instance_class = "db.t3.micro"     # Opcional, default: db.t3.micro
subnet_ids = []                       # Opcional, se obtienen automáticamente de la VPC
```

**Nota**: Si no proporcionas `db_password`, Terraform generará una automáticamente. Puedes verla en los outputs.

### Paso 2: Inicializar Terraform (si es primera vez)

```bash
cd terraform
terraform init
```

Esto descargará el provider `random` que se necesita para generar contraseñas.

### Paso 3: Plan y Apply

```bash
# Ver qué se va a crear
terraform plan

# Crear el RDS (esto puede tardar 5-10 minutos)
terraform apply
```

### Paso 4: Obtener Información del RDS

```bash
# Ver el endpoint del RDS
terraform output rds_endpoint

# Ver la dirección (sin puerto)
terraform output rds_address

# Ver el puerto
terraform output rds_port

# Ver el nombre de la base de datos
terraform output rds_db_name

# Ver el usuario (sensitive)
terraform output -raw rds_username

# Ver la contraseña generada (si no la proporcionaste)
terraform output -raw rds_password
```

### Paso 5: Actualizar env.production

Actualiza tu archivo `env.production` con los valores del RDS:

```bash
# En la raíz del proyecto
POSTGRES_HOST=<rds_address>           # Del output de terraform
POSTGRES_PORT=5432                    # Default
POSTGRES_USER=ewood_admin            # O el que configuraste
POSTGRES_PASSWORD=<password>          # La generada o la que pusiste
POSTGRES_DB=ewood_db
POSTGRES_SSLMODE=require              # Para producción
```

### Paso 6: Subir env.production a S3 y Desplegar

```bash
# Subir el archivo actualizado
source .env.push
./deploy.sh
```

## 🧪 Probar Localmente

Para probar localmente con PostgreSQL en Docker (ya configurado):

```bash
# Las variables ya están en docker-compose-local.yml
make dockerup

# Probar conexión
curl http://localhost:8200/health | jq
```

Deberías ver:
```json
{
  "status": "ok",
  "timestamp": "2025-11-30T...",
  "services": {
    "postgresql": "connected"
  }
}
```

## 🔍 Verificar en Producción

Después del despliegue:

```bash
# Obtener IP de EC2
cd terraform
EC2_IP=$(terraform output -raw ec2_public_ip)

# Probar health endpoint
curl http://$EC2_IP:8200/health | jq
```

## 📋 Variables de Entorno Requeridas

El código Go lee estas variables de entorno:

| Variable | Descripción | Local | Producción |
|----------|-------------|-------|------------|
| `POSTGRES_HOST` | Host de PostgreSQL | `postgres` | RDS endpoint |
| `POSTGRES_PORT` | Puerto (default: 5432) | `5432` | `5432` |
| `POSTGRES_USER` | Usuario | `ewood_user` | `ewood_admin` |
| `POSTGRES_PASSWORD` | Contraseña | `ewood_pass` | Generada por Terraform |
| `POSTGRES_DB` | Base de datos | `ewood_db` | `ewood_db` |
| `POSTGRES_SSLMODE` | Modo SSL | `disable` | `require` |

## 🔒 Seguridad

- ✅ RDS está en subnets privadas (no accesible públicamente)
- ✅ Security Group solo permite conexiones desde EC2
- ✅ Contraseña generada automáticamente si no se proporciona
- ✅ Encriptación de almacenamiento habilitada

## 💰 Costos

- `db.t3.micro`: ~$15/mes (instancia pequeña para pruebas)
- Almacenamiento: 20GB incluido, auto-scaling hasta 40GB
- Backup: 7 días de retención

## 🐛 Troubleshooting

### Error: "No subnets found"

**Causa**: La VPC no tiene suficientes subnets en diferentes AZs.

**Solución**: Proporciona `subnet_ids` manualmente en `terraform.tfvars`:
```hcl
subnet_ids = ["subnet-xxx", "subnet-yyy"]  # Al menos 2 en diferentes AZs
```

### Error: "Connection refused" en producción

**Causa**: Security Group no permite conexiones desde EC2.

**Solución**: Verifica que el Security Group de RDS permita conexiones desde el Security Group de EC2.

### Error: "password authentication failed"

**Causa**: Contraseña incorrecta en `env.production`.

**Solución**: Obtén la contraseña correcta:
```bash
terraform output -raw rds_password
```

## 📚 Próximos Pasos

1. ✅ Levantar RDS con Terraform
2. ✅ Actualizar `env.production` con credenciales
3. ✅ Desplegar y probar
4. ⭐ Crear tablas y esquemas
5. ⭐ Implementar migraciones
6. ⭐ Agregar modelos y repositorios

