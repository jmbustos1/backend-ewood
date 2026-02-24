# 📋 Resumen: Variables de Entorno vía S3

## 🎯 Lo que quieres hacer

Subir un archivo con variables de entorno (como `env.production`) a S3, igual que haces con `docker-compose.yml`, y que el contenedor lo use.

## ✅ Approach Simple

### 1. Crear archivo `env.production` (localmente)

```bash
# En la raíz del proyecto
ENV=production
PORT=:8080
MONGODB_URI=mongodb://...
MONGODB_DATABASE=ewood
POSTGRES_HOST=...
POSTGRES_USER=...
POSTGRES_PASSWORD=...
POSTGRES_DB=ewood_db
```

### 2. Agregar a `.gitignore`

```
env.production
```

### 3. Modificar `deploy.sh` para:

**A) Subir `env.production` a S3** (después de línea 47):
```bash
# 4.5 SUBIR ENV.PRODUCTION A S3
if [ -f env.production ]; then
    echo "📦 Subiendo env.production a S3..."
    aws s3 cp env.production "s3://$S3_BUCKET_NAME/env.production" --region "$AWS_REGION"
fi
```

**B) Descargarlo en EC2** (en el comando SSM, después de descargar docker-compose.yml):
```bash
\"aws s3 cp s3://$S3_BUCKET_NAME/env.production /home/ubuntu/env.production --region $AWS_REGION >> /tmp/deploy.log 2>&1 || echo 'env.production no encontrado' >> /tmp/deploy.log\",
```

### 4. Tu `docker-compose.yml` ya tiene `env_file` configurado

Solo necesitas asegurarte que apunte a:
```yaml
env_file:
  - /home/ubuntu/env.production
```

## 🔄 Flujo

1. Creas `env.production` localmente
2. Ejecutas `source .env.push && ./deploy.sh`
3. El script sube `env.production` a S3
4. El script descarga `env.production` en EC2
5. docker-compose lo carga con `env_file`
6. El contenedor arranca con las variables

## ❓ ¿Te parece bien este approach?

Es simple, reutiliza tu infraestructura S3, y es fácil de mantener. ¿Quieres que implemente los cambios en `deploy.sh` ahora?

