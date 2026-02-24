# 🔑 Obtener Credenciales del RDS

## 📋 Información del RDS Creado

Basado en los outputs de Terraform:

- **Endpoint**: `ewood-postgres.c3u0i2kuisp0.us-east-1.rds.amazonaws.com:5432`
- **Address**: `ewood-postgres.c3u0i2kuisp0.us-east-1.rds.amazonaws.com`
- **Port**: `5432`
- **Database**: `ewood_db`
- **Username**: `ewood_admin`

## 🔐 Obtener la Contraseña

La contraseña se generó automáticamente. Para obtenerla:

### Opción 1: Desde Terraform Output (Recomendado)

```bash
cd terraform

# Ver username
terraform output -raw rds_username

# Ver password (si el output está configurado)
terraform output -raw rds_password
```

### Opción 2: Desde el Estado de Terraform

```bash
cd terraform

# Ver la contraseña generada
terraform state show random_password.db_password | grep result
```

**Nota**: El valor estará marcado como `(sensitive value)`, pero puedes verlo con:

```bash
terraform state pull | jq -r '.resources[] | select(.type == "random_password" and .name == "db_password") | .instances[0].attributes.result'
```

### Opción 3: Desde AWS Console

1. Ve a AWS Console → RDS → Databases
2. Selecciona `ewood-postgres`
3. Ve a "Configuration" → "Manage master credentials"
4. Puedes ver el username, pero la contraseña no se muestra por seguridad

### Opción 4: Resetear la Contraseña (si no la tienes)

Si no puedes obtener la contraseña, puedes resetearla:

```bash
# 1. Generar nueva contraseña
NEW_PASSWORD=$(openssl rand -base64 16)

# 2. Actualizar en Terraform
# Edita terraform.tfvars y agrega:
# db_password = "tu_nueva_contraseña"

# 3. Aplicar cambios
terraform apply
```

## 📝 Actualizar env.production

Una vez que tengas las credenciales, actualiza `env.production`:

```bash
# En la raíz del proyecto
POSTGRES_HOST=ewood-postgres.c3u0i2kuisp0.us-east-1.rds.amazonaws.com
POSTGRES_PORT=5432
POSTGRES_USER=ewood_admin
POSTGRES_PASSWORD=<la_contraseña_obtenida>
POSTGRES_DB=ewood_db
POSTGRES_SSLMODE=require
```

## 🔍 Verificar Conexión

### Desde tu máquina local (si tienes psql)

```bash
psql -h ewood-postgres.c3u0i2kuisp0.us-east-1.rds.amazonaws.com \
     -p 5432 \
     -U ewood_admin \
     -d ewood_db
```

### Desde EC2 (vía SSM)

```bash
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["psql -h ewood-postgres.c3u0i2kuisp0.us-east-1.rds.amazonaws.com -p 5432 -U ewood_admin -d ewood_db -c \"SELECT version();\""]'
```

## ⚠️ Importante

- **Guarda la contraseña de forma segura** (password manager, etc.)
- **No commitees** la contraseña en el repositorio
- **Rota la contraseña** periódicamente en producción

