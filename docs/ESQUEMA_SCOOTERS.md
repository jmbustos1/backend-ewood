# 🛴 Esquema de Base de Datos para Scooters

## 📊 Resumen del Diseño

Este esquema está diseñado para **guardar el estado actual de los scooters** y **datos operacionales importantes**, pero **NO guarda telemetría raw** (medidas de cada segundo).

### ✅ Lo que SÍ guardamos:
- **Estado actual** del scooter (última posición, batería, estado)
- **Historial de viajes** (inicio, fin, métricas agregadas)
- **Ubicaciones clave** (inicio de viaje, fin de viaje, cambios de estado)
- **Historial de cambios de estado**

### ❌ Lo que NO guardamos:
- Telemetría raw (GPS cada segundo durante el viaje)
- Datos de sensores en tiempo real
- Historial completo de ubicaciones durante el viaje

> 💡 **Nota**: Si en el futuro necesitas telemetría raw, considera usar un sistema de time-series como **TimescaleDB** (extensión de PostgreSQL) o **InfluxDB**.

---

## 🗄️ Estructura de Tablas

### 1. `scooters` - Estado Actual de Scooters

Guarda el **estado actual** de cada scooter (última posición, batería, estado).

```sql
scooters
├── id (PK)
├── code (UNIQUE) - Código único del scooter
├── model - Modelo del scooter
├── status - Estado actual: available, in_use, maintenance, offline
├── battery_level - Porcentaje de batería (0-100)
├── last_latitude - Última latitud conocida
├── last_longitude - Última longitud conocida
├── last_location_updated_at - Cuándo se actualizó la ubicación
├── current_user_id - Usuario que está usando el scooter (NULL si disponible)
├── current_trip_id - ID del viaje actual (NULL si no está en viaje)
├── trip_started_at - Cuándo empezó el viaje actual
├── created_at
└── updated_at
```

**Uso típico:**
- Consultar scooters disponibles cerca de una ubicación
- Ver estado actual de un scooter específico
- Actualizar última posición cuando el scooter reporta ubicación

---

### 2. `trips` - Historial de Viajes

Guarda el **historial de viajes** con métricas agregadas, NO telemetría raw.

```sql
trips
├── id (PK)
├── scooter_id (FK) - Scooter usado
├── user_id (FK) - Usuario que hizo el viaje
├── start_latitude - Latitud de inicio
├── start_longitude - Longitud de inicio
├── end_latitude - Latitud de fin (NULL si está activo)
├── end_longitude - Longitud de fin (NULL si está activo)
├── started_at - Cuándo empezó el viaje
├── ended_at - Cuándo terminó (NULL si está activo)
├── duration_seconds - Duración en segundos
├── distance_meters - Distancia total (calculada)
├── average_speed_kmh - Velocidad promedio
├── max_speed_kmh - Velocidad máxima
├── status - active, completed, cancelled
├── cost_amount - Costo del viaje
├── created_at
└── updated_at
```

**Uso típico:**
- Ver historial de viajes de un usuario
- Ver historial de viajes de un scooter
- Calcular estadísticas (ingresos, distancias, etc.)

---

### 3. `scooter_locations` - Ubicaciones Históricas (Puntos Clave)

Guarda **solo ubicaciones importantes**, no cada segundo.

```sql
scooter_locations
├── id (PK)
├── scooter_id (FK)
├── trip_id (FK, nullable) - Si es parte de un viaje
├── latitude
├── longitude
├── event_type - trip_start, trip_end, status_change, periodic_update
├── battery_level - Batería en ese momento
├── speed_kmh - Velocidad en ese momento
└── recorded_at
```

**Eventos que se guardan:**
- `trip_start` - Inicio de viaje
- `trip_end` - Fin de viaje
- `status_change` - Cambio de estado (ej: available → maintenance)
- `periodic_update` - Actualización periódica (ej: cada 30 segundos durante viaje activo)

**Uso típico:**
- Trazar ruta de un viaje (usando solo puntos clave)
- Ver última ubicación conocida
- Análisis de patrones de movimiento

---

### 4. `scooter_status_history` - Historial de Cambios de Estado

Guarda el **historial de cambios de estado** del scooter.

```sql
scooter_status_history
├── id (PK)
├── scooter_id (FK)
├── old_status - Estado anterior
├── new_status - Estado nuevo
├── reason - Razón del cambio
└── changed_at
```

**Uso típico:**
- Auditoría de cambios de estado
- Análisis de tiempo en cada estado
- Debugging de problemas

---

## 🔄 Flujo de Actualización

### Cuando un scooter reporta su ubicación:

```sql
-- 1. Actualizar estado actual del scooter
UPDATE scooters 
SET 
    last_latitude = $1,
    last_longitude = $2,
    last_location_updated_at = NOW(),
    battery_level = $3
WHERE id = $4;

-- 2. Si está en viaje, guardar ubicación clave (cada 30 segundos, por ejemplo)
INSERT INTO scooter_locations (scooter_id, trip_id, latitude, longitude, event_type, battery_level)
VALUES ($scooter_id, $trip_id, $lat, $lng, 'periodic_update', $battery);
```

### Cuando un usuario inicia un viaje:

```sql
-- 1. Crear registro de viaje
INSERT INTO trips (scooter_id, user_id, start_latitude, start_longitude, status)
VALUES ($scooter_id, $user_id, $lat, $lng, 'active');

-- 2. Actualizar scooter
UPDATE scooters 
SET 
    status = 'in_use',
    current_user_id = $user_id,
    current_trip_id = $trip_id,
    trip_started_at = NOW()
WHERE id = $scooter_id;

-- 3. Guardar ubicación de inicio
INSERT INTO scooter_locations (scooter_id, trip_id, latitude, longitude, event_type)
VALUES ($scooter_id, $trip_id, $lat, $lng, 'trip_start');

-- 4. Registrar cambio de estado
INSERT INTO scooter_status_history (scooter_id, old_status, new_status, reason)
VALUES ($scooter_id, 'available', 'in_use', 'user_started_trip');
```

### Cuando un usuario termina un viaje:

```sql
-- 1. Actualizar viaje
UPDATE trips 
SET 
    end_latitude = $lat,
    end_longitude = $lng,
    ended_at = NOW(),
    duration_seconds = EXTRACT(EPOCH FROM (NOW() - started_at)),
    distance_meters = $distance,
    average_speed_kmh = $avg_speed,
    max_speed_kmh = $max_speed,
    status = 'completed',
    cost_amount = $cost
WHERE id = $trip_id;

-- 2. Actualizar scooter
UPDATE scooters 
SET 
    status = 'available',
    current_user_id = NULL,
    current_trip_id = NULL,
    trip_started_at = NULL,
    last_latitude = $lat,
    last_longitude = $lng
WHERE id = $scooter_id;

-- 3. Guardar ubicación de fin
INSERT INTO scooter_locations (scooter_id, trip_id, latitude, longitude, event_type)
VALUES ($scooter_id, $trip_id, $lat, $lng, 'trip_end');

-- 4. Registrar cambio de estado
INSERT INTO scooter_status_history (scooter_id, old_status, new_status, reason)
VALUES ($scooter_id, 'in_use', 'available', 'user_ended_trip');
```

---

## 📊 Consultas Útiles

### Scooters disponibles cerca de una ubicación:

```sql
SELECT 
    id, code, battery_level, 
    last_latitude, last_longitude,
    (6371 * acos(
        cos(radians($user_lat)) * 
        cos(radians(last_latitude)) * 
        cos(radians(last_longitude) - radians($user_lng)) + 
        sin(radians($user_lat)) * 
        sin(radians(last_latitude))
    )) AS distance_km
FROM scooters
WHERE status = 'available'
  AND last_latitude IS NOT NULL
  AND last_longitude IS NOT NULL
  AND battery_level > 20 -- Mínimo 20% de batería
ORDER BY distance_km
LIMIT 20;
```

### Estado actual de un scooter:

```sql
SELECT 
    s.*,
    u.email AS current_user_email,
    t.started_at AS trip_started_at
FROM scooters s
LEFT JOIN users u ON s.current_user_id = u.id
LEFT JOIN trips t ON s.current_trip_id = t.id
WHERE s.id = $scooter_id;
```

### Historial de viajes de un usuario:

```sql
SELECT 
    t.*,
    s.code AS scooter_code
FROM trips t
JOIN scooters s ON t.scooter_id = s.id
WHERE t.user_id = $user_id
ORDER BY t.started_at DESC;
```

### Ruta de un viaje (usando puntos clave):

```sql
SELECT 
    latitude, longitude, recorded_at, event_type
FROM scooter_locations
WHERE trip_id = $trip_id
ORDER BY recorded_at;
```

---

## 🎯 Ventajas de este Diseño

1. **Eficiente**: No guarda datos innecesarios
2. **Rápido**: Consultas rápidas sobre estado actual
3. **Escalable**: Puede manejar muchos scooters sin problemas
4. **Flexible**: Fácil agregar más campos si es necesario
5. **Auditable**: Historial de cambios de estado

---

## 🚀 Próximos Pasos

1. **Ejecutar migración**: `make migrate-up`
2. **Crear modelos GORM** para las tablas
3. **Crear servicios** para gestionar scooters
4. **Crear handlers** para API de scooters
5. **Implementar WebSocket** para actualizaciones en tiempo real

---

## 💡 Notas Importantes

- **Telemetría raw**: Si en el futuro necesitas guardar cada segundo de GPS, considera:
  - **TimescaleDB** (extensión de PostgreSQL para time-series)
  - **InfluxDB** (base de datos especializada en time-series)
  - **S3 + Parquet** (para análisis histórico)

- **Índices espaciales**: Si necesitas búsquedas geográficas más complejas, considera usar **PostGIS** (extensión de PostgreSQL para datos geoespaciales).

