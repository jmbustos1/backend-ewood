# 🗄️ Configuración de Bases de Datos Locales

Este documento describe cómo usar MongoDB y PostgreSQL en el entorno de desarrollo local.

## 📋 Información de Conexión

### MongoDB

- **Puerto**: `27017`
- **Usuario root**: `admin`
- **Contraseña root**: `admin123`
- **Base de datos**: `ewood`
- **URI de conexión**: `mongodb://admin:admin123@localhost:27017/ewood?authSource=admin`

#### Conectar desde la terminal:

```bash
docker exec -it ewood-mongodb mongosh -u admin -p admin123 --authenticationDatabase admin
```

O desde el host:

```bash
mongosh "mongodb://admin:admin123@localhost:27017/ewood?authSource=admin"
```

### PostgreSQL

- **Puerto**: `5432`
- **Usuario**: `ewood_user`
- **Contraseña**: `ewood_pass`
- **Base de datos**: `ewood_db`
- **URI de conexión**: `postgresql://ewood_user:ewood_pass@localhost:5432/ewood_db?sslmode=disable`

#### Conectar desde la terminal:

```bash
docker exec -it ewood-postgres psql -U ewood_user -d ewood_db
```

O desde el host:

```bash
psql -h localhost -p 5432 -U ewood_user -d ewood_db
# Contraseña: ewood_pass
```

## 🚀 Iniciar las Bases de Datos

Para levantar todas las bases de datos junto con el servicio Go:

```bash
make dockerup
```

O directamente:

```bash
docker compose -f docker-compose-local.yml up
```

## 🛑 Detener las Bases de Datos

```bash
make dockerdown
```

Para detener y eliminar los volúmenes (⚠️ **Borra todos los datos**):

```bash
make dropdb-and-container
```

## 🔌 Variables de Entorno en el Contenedor Go

El servicio `golang-ewood` tiene configuradas automáticamente las siguientes variables de entorno:

### MongoDB
- `MONGODB_URI=mongodb://admin:admin123@mongodb:27017/ewood?authSource=admin`
- `MONGODB_DATABASE=ewood`

### PostgreSQL
- `POSTGRES_HOST=postgres`
- `POSTGRES_PORT=5432`
- `POSTGRES_USER=ewood_user`
- `POSTGRES_PASSWORD=ewood_pass`
- `POSTGRES_DB=ewood_db`
- `POSTGRES_SSLMODE=disable`

**Nota**: Dentro del contenedor, usa los nombres de servicio (`mongodb` y `postgres`) como hosts. Desde el host local, usa `localhost`.

## 💾 Persistencia de Datos

Los datos se almacenan en volúmenes Docker nombrados:
- `mongodb_data`: Datos de MongoDB
- `mongodb_config`: Configuración de MongoDB
- `postgres_data`: Datos de PostgreSQL

Estos volúmenes persisten aunque detengas los contenedores. Para eliminarlos:

```bash
docker compose -f docker-compose-local.yml down -v
```

## 🔍 Verificar Estado de las Bases de Datos

### MongoDB

```bash
# Ver logs
docker logs ewood-mongodb

# Verificar salud
docker exec ewood-mongodb mongosh --eval "db.adminCommand('ping')"
```

### PostgreSQL

```bash
# Ver logs
docker logs ewood-postgres

# Verificar salud
docker exec ewood-postgres pg_isready -U ewood_user -d ewood_db
```

## 📝 Ejemplos de Uso en Go

### MongoDB

```go
package main

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func connectMongoDB() (*mongo.Client, error) {
    uri := os.Getenv("MONGODB_URI")
    clientOptions := options.Client().ApplyURI(uri)
    
    client, err := mongo.Connect(context.TODO(), clientOptions)
    if err != nil {
        return nil, err
    }
    
    err = client.Ping(context.TODO(), nil)
    if err != nil {
        return nil, err
    }
    
    return client, nil
}
```

### PostgreSQL

```go
package main

import (
    "database/sql"
    _ "github.com/lib/pq"
)

func connectPostgreSQL() (*sql.DB, error) {
    connStr := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        os.Getenv("POSTGRES_HOST"),
        os.Getenv("POSTGRES_PORT"),
        os.Getenv("POSTGRES_USER"),
        os.Getenv("POSTGRES_PASSWORD"),
        os.Getenv("POSTGRES_DB"),
        os.Getenv("POSTGRES_SSLMODE"),
    )
    
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    
    err = db.Ping()
    if err != nil {
        return nil, err
    }
    
    return db, nil
}
```

## 📊 Gestión de Logs

### ¿Qué significan esos logs de MongoDB?

Cuando ejecutas `docker compose up`, verás muchos logs de MongoDB que son **completamente normales**. Estos logs muestran:

- **"Connection accepted"**: Se aceptó una nueva conexión
- **"client metadata"**: Información sobre el cliente que se conecta (mongosh, drivers, etc.)
- **"Connection not authenticating"**: Conexión sin autenticación inicial (normal para conexiones locales)

Estos logs aparecen porque MongoDB registra cada conexión y actividad. Es normal verlos especialmente durante:
- Healthchecks de Docker
- Conexiones de herramientas de administración
- Conexiones de tu aplicación

### Ver solo logs de un servicio específico

Para ver solo los logs del servicio Go (sin el ruido de las bases de datos):

```bash
docker compose -f docker-compose-local.yml logs -f golang-ewood
```

Para ver solo MongoDB:

```bash
docker compose -f docker-compose-local.yml logs -f mongodb
```

Para ver solo PostgreSQL:

```bash
docker compose -f docker-compose-local.yml logs -f postgres
```

### Ver logs sin seguimiento (últimas líneas)

```bash
# Últimas 50 líneas del servicio Go
docker compose -f docker-compose-local.yml logs --tail=50 golang-ewood

# Últimas 100 líneas de todos los servicios
docker compose -f docker-compose-local.yml logs --tail=100
```

### Ejecutar en modo silencioso (detached)

Para ejecutar los servicios sin ver los logs en tiempo real:

```bash
docker compose -f docker-compose-local.yml up -d
```

Luego ver los logs cuando los necesites:

```bash
docker compose -f docker-compose-local.yml logs -f
```

### Configuración de logs

Los servicios están configurados con límites de logs:
- Tamaño máximo por archivo: 10MB
- Máximo de archivos rotados: 3
- Esto previene que los logs ocupen demasiado espacio en disco

## 🔒 Seguridad

**⚠️ IMPORTANTE**: Las credenciales en `docker-compose-local.yml` son solo para desarrollo local. 

Para producción:
- Usa variables de entorno o un gestor de secretos (AWS Secrets Manager, etc.)
- No commitees credenciales reales en el repositorio
- Cambia las contraseñas por defecto antes de usar en un entorno compartido

