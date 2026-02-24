# 📦 Configuración de Variables de Entorno vía S3

## 🎯 Approach Simple

Siguiendo el mismo patrón que ya usas con `docker-compose.yml`:
1. Crear archivo `env.production` con las variables
2. Subirlo a S3
3. Descargarlo en EC2 antes de iniciar docker-compose
4. docker-compose.yml ya tiene `env_file` configurado

## 📝 Paso 1: Crear archivo env.production

Crea un archivo `env.production` en la raíz del proyecto:

```bash
# Configuración del servidor
ENV=production
PORT=:8080

# MongoDB (cuando tengas las bases de datos)
MONGODB_URI=mongodb://usuario:contraseña@host:27017/ewood?authSource=admin
MONGODB_DATABASE=ewood

# PostgreSQL (cuando tengas las bases de datos)
POSTGRES_HOST=tu-postgres-host.rds.amazonaws.com
POSTGRES_PORT=5432
POSTGRES_USER=ewood_user
POSTGRES_PASSWORD=tu_contraseña_segura
POSTGRES_DB=ewood_db
POSTGRES_SSLMODE=require
```

## 🔒 Paso 2: Agregar a .gitignore

Asegúrate de que `env.production` esté en `.gitignore`:

```bash
# Variables de entorno de producción
env.production
.env.production
```

## 📤 Paso 3: Modificar deploy.sh para subir a S3

Agregar al script `deploy.sh` después de subir docker-compose.yml:

```bash
# 4.5 SUBIR ENV.PRODUCTION A S3 (si existe)
if [ -f env.production ]; then
    echo "📦 Subiendo env.production a S3..."
    aws s3 cp env.production "s3://$S3_BUCKET_NAME/env.production" --region "$AWS_REGION"
else
    echo "⚠️  Advertencia: env.production no existe, se usará configuración por defecto"
fi
```

## 📥 Paso 4: Modificar comando SSM para descargar

En el comando SSM de `deploy.sh`, agregar después de descargar docker-compose.yml:

```bash
\"aws s3 cp s3://$S3_BUCKET_NAME/env.production /home/ubuntu/env.production --region $AWS_REGION >> /tmp/deploy.log 2>&1 || echo 'env.production no encontrado en S3' >> /tmp/deploy.log\",
```

## 🐳 Paso 5: Ajustar docker-compose.yml

Ya tienes `env_file` configurado, solo asegúrate de que apunte a la ruta correcta:

```yaml
env_file:
  - /home/ubuntu/env.production
```

## ✅ Flujo Completo

1. **Local**: Creas `env.production` con tus variables
2. **Local**: Ejecutas `source .env.push && ./deploy.sh`
   - El script sube `env.production` a S3
3. **EC2**: El script descarga `env.production` desde S3
4. **EC2**: docker-compose carga las variables desde el archivo
5. **EC2**: El contenedor arranca con las variables configuradas

## 🔍 Ventajas de este Approach

- ✅ Simple y directo
- ✅ Reutiliza tu infraestructura S3 existente
- ✅ No necesitas Secrets Manager (por ahora)
- ✅ Fácil de actualizar: solo cambias el archivo y vuelves a subir
- ✅ Mismo patrón que docker-compose.yml

## ⚠️ Consideraciones de Seguridad

- **Nunca** commitees `env.production` (ya está en .gitignore)
- Las credenciales están en texto plano en S3 (considera encriptar el bucket)
- Para mayor seguridad, más adelante puedes migrar a Secrets Manager

## 🚀 Próximos Pasos

1. Crear `env.production` con tus variables
2. Modificar `deploy.sh` para subirlo a S3
3. Probar el despliegue

