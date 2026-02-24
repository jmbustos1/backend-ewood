# 🚀 Estrategia de Despliegue con Migraciones

Este documento describe la estrategia completa para desplegar la aplicación con migraciones de base de datos de forma idempotente y segura.

## 📋 Resumen

El sistema de despliegue integra automáticamente las migraciones de base de datos en el proceso de deployment, asegurando que:

1. ✅ Las migraciones se ejecuten automáticamente después de cada despliegue
2. ✅ El proceso sea idempotente (puede ejecutarse múltiples veces sin problemas)
3. ✅ Los errores de migración no bloqueen el despliegue (se registran como advertencias)
4. ✅ Las migraciones estén versionadas y puedan revertirse

---

## 🔄 Flujo de Despliegue

### 1. Build y Push

```bash
# Construir imagen Docker (incluye código + migraciones)
docker build -t ewood-backend .

# Subir a ECR
docker push <ECR_URI>
```

**Lo que se incluye en la imagen:**
- ✅ Binario del servidor (`/app/server`)
- ✅ Binario de migraciones (`/app/migrate`)
- ✅ Archivos de migración SQL (`/app/migrations/`)

### 2. Despliegue en EC2

El script `deploy.sh` ejecuta automáticamente:

1. Descarga `docker-compose.yml` y `env.production` desde S3
2. Hace login en ECR
3. Pull de la nueva imagen
4. Detiene el contenedor anterior
5. **Levanta el nuevo contenedor**
6. **Ejecuta migraciones pendientes** ⭐
7. Registra el resultado en logs

### 3. Ejecución de Migraciones

Las migraciones se ejecutan automáticamente después de levantar el contenedor:

```bash
docker exec ewood-golang-container /app/migrate -command=up -path=/app/migrations
```

**Características:**
- ✅ Se ejecuta dentro del contenedor (mismo entorno que la app)
- ✅ Usa las mismas credenciales de DB que la aplicación
- ✅ Si no hay migraciones pendientes, retorna exitosamente
- ✅ Si hay error, se registra pero no bloquea el despliegue

---

## 🎯 Estrategia por Ambiente

### Desarrollo Local

```bash
# 1. Levantar servicios (DB + App)
make dockerup

# 2. Ejecutar migraciones manualmente
make migrate-up

# 3. Verificar versión
make migrate-version
```

### Staging/Testing

1. Desplegar con `deploy.sh`
2. Las migraciones se ejecutan automáticamente
3. Verificar logs:
   ```bash
   aws ssm send-command \
     --document-name "AWS-RunShellScript" \
     --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"i-xxxxx\"]}]" \
     --parameters 'commands=["docker logs ewood-golang-container | grep -i migrate"]'
   ```

### Producción

1. **Pre-despliegue:**
   - ✅ Backup de la base de datos
   - ✅ Revisar migraciones pendientes
   - ✅ Verificar que el rollback funcione

2. **Despliegue:**
   ```bash
   source .env.push && ./deploy.sh
   ```

3. **Post-despliegue:**
   - ✅ Verificar logs de migraciones
   - ✅ Verificar versión de migraciones
   - ✅ Probar endpoints críticos

---

## 🔧 Comandos Útiles

### Verificar Estado de Migraciones en Producción

```bash
# Ver versión actual
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["docker exec ewood-golang-container /app/migrate -command=version -path=/app/migrations"]'
```

### Ejecutar Migraciones Manualmente en Producción

```bash
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["docker exec ewood-golang-container /app/migrate -command=up -path=/app/migrations"]'
```

### Ver Logs de Despliegue

```bash
# Ver logs completos del último despliegue
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["cat /tmp/deploy.log"]'
```

---

## ⚠️ Manejo de Errores

### Error en Migraciones Durante Despliegue

Si una migración falla durante el despliegue:

1. **El despliegue continúa** (no se bloquea)
2. **Se registra una advertencia** en los logs
3. **El contenedor queda corriendo** con la versión anterior del código

**Acción requerida:**
1. Revisar logs para identificar el error
2. Corregir el problema (migración o base de datos)
3. Ejecutar migraciones manualmente o hacer rollback
4. Redesplegar si es necesario

### Estado "Dirty" de Migraciones

Si una migración falla a mitad de camino, la base de datos queda en estado "dirty".

**Solución:**
1. Identificar qué migración falló
2. Corregir manualmente el estado de la base de datos
3. Limpiar el estado dirty:
   ```sql
   UPDATE schema_migrations SET dirty = false;
   ```
4. Reintentar la migración

---

## 📊 Monitoreo

### Métricas a Monitorear

1. **Tiempo de ejecución de migraciones**: Debe ser razonable (< 1 min para migraciones normales)
2. **Versión de migraciones**: Debe coincidir con la versión esperada
3. **Errores de migración**: Deben ser cero en producción

### Alertas Recomendadas

- ⚠️ Migración fallida durante despliegue
- ⚠️ Estado "dirty" en base de datos
- ⚠️ Tiempo de migración > 5 minutos

---

## 🔐 Seguridad

### Credenciales

- Las credenciales de DB vienen de `env.production` (almacenado en S3)
- Las migraciones usan las mismas credenciales que la aplicación
- Las migraciones tienen acceso completo a la base de datos

### Buenas Prácticas

1. ✅ Nunca modifiques migraciones ya aplicadas
2. ✅ Siempre prueba el rollback localmente
3. ✅ Haz backup antes de migraciones grandes
4. ✅ Revisa las migraciones antes de desplegar
5. ✅ Usa transacciones cuando sea posible

---

## 🚨 Rollback de Despliegue

Si necesitas revertir un despliegue completo:

### Opción 1: Rollback de Migración + Redesplegar Versión Anterior

```bash
# 1. Revertir última migración
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"$EC2_INSTANCE_ID\"]}]" \
  --parameters 'commands=["docker exec ewood-golang-container /app/migrate -command=down -path=/app/migrations"]'

# 2. Redesplegar versión anterior del código
# (usar IMAGE_TAG de la versión anterior)
```

### Opción 2: Restaurar desde Backup

1. Restaurar base de datos desde backup
2. Redesplegar versión anterior del código

---

## 📝 Checklist de Despliegue

Antes de desplegar:

- [ ] Migraciones probadas localmente
- [ ] Rollback probado localmente
- [ ] Backup de base de datos (producción)
- [ ] Revisión de código de migraciones
- [ ] Variables de entorno actualizadas
- [ ] Logs de despliegue monitoreados

Después de desplegar:

- [ ] Verificar que las migraciones se ejecutaron
- [ ] Verificar versión de migraciones
- [ ] Probar endpoints críticos
- [ ] Revisar logs de aplicación
- [ ] Monitorear métricas

---

## 🔗 Referencias

- [docs/MIGRACIONES.md](MIGRACIONES.md) - Guía completa de migraciones
- [docs/DEPLOY.md](DEPLOY.md) - Guía de despliegue general
- [golang-migrate](https://github.com/golang-migrate/migrate) - Documentación oficial

