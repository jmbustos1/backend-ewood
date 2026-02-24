# 🗄️ Sistema de Migraciones

Este proyecto utiliza **golang-migrate** para gestionar las migraciones de base de datos de forma versionada y controlada.

## 📋 Índice

- [Estructura de Migraciones](#estructura-de-migraciones)
- [Comandos Disponibles](#comandos-disponibles)
- [Crear una Nueva Migración](#crear-una-nueva-migración)
- [Estrategia de Despliegue](#estrategia-de-despliegue)
- [Troubleshooting](#troubleshooting)

---

## 📁 Estructura de Migraciones

Las migraciones se encuentran en el directorio `migrations/` y siguen el formato:

```
migrations/
├── 000001_initial_schema.up.sql    # Migración hacia adelante
├── 000001_initial_schema.down.sql   # Rollback de la migración
├── 000002_add_users_table.up.sql
├── 000002_add_users_table.down.sql
└── ...
```

**Convención de nombres:**
- `{número}_{descripción}.up.sql` - Migración hacia adelante
- `{número}_{descripción}.down.sql` - Rollback de la migración
- Los números deben ser secuenciales y únicos

---

## 🛠️ Comandos Disponibles

### Desarrollo Local

#### Ejecutar migraciones pendientes
```bash
make migrate-up
# o directamente:
go run ./cmd/migrate -command=up
```

#### Revertir última migración
```bash
make migrate-down
# o directamente:
go run ./cmd/migrate -command=down
```

#### Ver versión actual
```bash
make migrate-version
# o directamente:
go run ./cmd/migrate -command=version
```

### En Contenedor Docker Local

```bash
# Ejecutar migraciones en el contenedor
make migrate-up-docker

# Ver versión en el contenedor
make migrate-version-docker
```

### En Producción

Las migraciones se ejecutan automáticamente después de cada despliegue. El script `deploy.sh` incluye un paso que ejecuta las migraciones pendientes.

Si necesitas ejecutar migraciones manualmente en producción:

```bash
# Conectarse a la instancia EC2 vía SSM
aws ssm send-command \
  --document-name "AWS-RunShellScript" \
  --targets "[{\"Key\":\"InstanceIds\",\"Values\":[\"i-xxxxx\"]}]" \
  --parameters 'commands=["docker exec ewood-golang-container /app/migrate -command=up -path=/app/migrations"]'
```

---

## ➕ Crear una Nueva Migración

### Paso 1: Crear los archivos de migración

Crea dos archivos en `migrations/`:

**`migrations/000002_add_products_table.up.sql`**
```sql
-- Crear tabla de productos
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
```

**`migrations/000002_add_products_table.down.sql`**
```sql
-- Rollback: eliminar tabla de productos
DROP TABLE IF EXISTS products;
```

### Paso 2: Ejecutar la migración

```bash
make migrate-up
```

### Paso 3: Verificar

```bash
make migrate-version
```

---

## 🚀 Estrategia de Despliegue

### Flujo Automático

1. **Build**: Se construye la imagen Docker con el código y las migraciones
2. **Push**: Se sube la imagen a ECR
3. **Deploy**: Se despliega en EC2
4. **Migrate**: Se ejecutan automáticamente las migraciones pendientes

### Migraciones en el Despliegue

El script `deploy.sh` incluye automáticamente la ejecución de migraciones:

```bash
# Después de levantar el contenedor
docker exec ewood-golang-container /app/migrate -command=up -path=/app/migrations
```

### Estrategia Recomendada

1. **Desarrollo Local**: 
   - Crear y probar migraciones localmente
   - Verificar que el rollback funcione correctamente

2. **Staging/Testing**:
   - Probar las migraciones en un ambiente similar a producción
   - Verificar tiempos de ejecución

3. **Producción**:
   - Las migraciones se ejecutan automáticamente en cada despliegue
   - Monitorear los logs del despliegue
   - Si hay error, el despliegue continúa pero se registra una advertencia

### Migraciones Críticas

Para migraciones que requieren atención especial (ej: cambios de esquema grandes):

1. **Backup**: Asegúrate de tener un backup de la base de datos
2. **Ventana de Mantenimiento**: Considera ejecutar en horarios de bajo tráfico
3. **Monitoreo**: Supervisa el proceso de migración
4. **Rollback Plan**: Ten un plan claro de rollback si algo sale mal

---

## 🔧 Troubleshooting

### Error: "no change"

```
ℹ️  No hay migraciones pendientes
```

Esto es normal cuando todas las migraciones ya están aplicadas.

### Error: "dirty database version"

Si ves este error, significa que una migración anterior falló a mitad de camino.

**Solución:**
1. Revisar los logs para identificar qué migración falló
2. Corregir el problema en la base de datos manualmente
3. Marcar la versión como limpia:
   ```sql
   UPDATE schema_migrations SET dirty = false;
   ```

### Error: "connection refused"

Asegúrate de que:
- Las variables de entorno de PostgreSQL estén configuradas
- La base de datos esté accesible
- El contenedor esté corriendo (si usas Docker)

### Verificar Estado de Migraciones

```bash
# Ver versión actual
make migrate-version

# Ver logs del contenedor
docker logs ewood-golang-container
```

---

## 📚 Recursos Adicionales

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [GORM Documentation](https://gorm.io/docs/)
- [PostgreSQL Migrations Best Practices](https://www.postgresql.org/docs/current/ddl-alter.html)

---

## 💡 Mejores Prácticas

1. **Siempre crea archivos `.down.sql`**: Permite revertir cambios si es necesario
2. **Prueba el rollback localmente**: Antes de desplegar, verifica que el rollback funcione
3. **Nombres descriptivos**: Usa nombres claros que indiquen qué hace la migración
4. **Una migración, un cambio**: Evita hacer múltiples cambios no relacionados en una sola migración
5. **Versiona las migraciones**: Nunca modifiques una migración ya aplicada, crea una nueva
6. **Backups antes de migraciones grandes**: Siempre haz backup antes de cambios importantes

---

## 🔐 Seguridad

- Las migraciones se ejecutan con las credenciales de la base de datos configuradas
- En producción, las credenciales vienen de `env.production` (almacenado en S3)
- Las migraciones tienen acceso completo a la base de datos, úsalas con cuidado

