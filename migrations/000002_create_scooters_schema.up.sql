-- Migración: Esquema de Scooters
-- Guarda el estado actual de los scooters (última posición, batería, estado)
-- NO guarda telemetría raw (solo la última medida)

-- Tabla de scooters (estado actual)
CREATE TABLE IF NOT EXISTS scooters (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE, -- Código único del scooter (ej: "SCO-001")
    model VARCHAR(100), -- Modelo del scooter
    status VARCHAR(20) NOT NULL DEFAULT 'available', -- available, in_use, maintenance, offline
    battery_level INTEGER NOT NULL DEFAULT 100 CHECK (battery_level >= 0 AND battery_level <= 100), -- Porcentaje de batería (0-100)
    
    -- Última ubicación conocida
    last_latitude DECIMAL(10, 8), -- Última latitud
    last_longitude DECIMAL(11, 8), -- Última longitud
    last_location_updated_at TIMESTAMP, -- Cuándo se actualizó la última ubicación
    
    -- Información del viaje actual (si está en uso)
    current_user_id INTEGER, -- Usuario que está usando el scooter (NULL si está disponible)
    current_trip_id INTEGER, -- ID del viaje actual (NULL si no está en viaje)
    trip_started_at TIMESTAMP, -- Cuándo empezó el viaje actual
    
    -- Metadatos
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Índices
    CONSTRAINT fk_current_user FOREIGN KEY (current_user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Índices para búsquedas rápidas
CREATE INDEX IF NOT EXISTS idx_scooters_status ON scooters(status);
CREATE INDEX IF NOT EXISTS idx_scooters_code ON scooters(code);
CREATE INDEX IF NOT EXISTS idx_scooters_battery ON scooters(battery_level);
CREATE INDEX IF NOT EXISTS idx_scooters_location ON scooters(last_latitude, last_longitude) WHERE last_latitude IS NOT NULL AND last_longitude IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scooters_current_user ON scooters(current_user_id) WHERE current_user_id IS NOT NULL;

-- Tabla de viajes/trips (historial de viajes, sin telemetría raw)
CREATE TABLE IF NOT EXISTS trips (
    id SERIAL PRIMARY KEY,
    scooter_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    
    -- Ubicaciones del viaje (solo inicio y fin, no cada segundo)
    start_latitude DECIMAL(10, 8) NOT NULL,
    start_longitude DECIMAL(11, 8) NOT NULL,
    end_latitude DECIMAL(10, 8),
    end_longitude DECIMAL(11, 8),
    
    -- Tiempos
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    duration_seconds INTEGER, -- Duración en segundos (calculado)
    
    -- Métricas del viaje (agregadas, no raw)
    distance_meters DECIMAL(10, 2), -- Distancia total en metros
    average_speed_kmh DECIMAL(5, 2), -- Velocidad promedio en km/h
    max_speed_kmh DECIMAL(5, 2), -- Velocidad máxima en km/h
    
    -- Estado
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, completed, cancelled
    cost_amount DECIMAL(10, 2), -- Costo del viaje
    
    -- Metadatos
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    CONSTRAINT fk_trip_scooter FOREIGN KEY (scooter_id) REFERENCES scooters(id) ON DELETE RESTRICT,
    CONSTRAINT fk_trip_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Índices para trips
CREATE INDEX IF NOT EXISTS idx_trips_scooter ON trips(scooter_id);
CREATE INDEX IF NOT EXISTS idx_trips_user ON trips(user_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips(status);
CREATE INDEX IF NOT EXISTS idx_trips_started_at ON trips(started_at);
CREATE INDEX IF NOT EXISTS idx_trips_user_active ON trips(user_id, status) WHERE status = 'active';

-- Tabla de ubicaciones históricas (solo puntos clave, no cada segundo)
-- Guarda ubicaciones importantes: inicio de viaje, fin de viaje, cambios de estado
CREATE TABLE IF NOT EXISTS scooter_locations (
    id SERIAL PRIMARY KEY,
    scooter_id INTEGER NOT NULL,
    trip_id INTEGER, -- NULL si no es parte de un viaje
    
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    
    -- Contexto de la ubicación
    event_type VARCHAR(50) NOT NULL, -- trip_start, trip_end, status_change, periodic_update
    battery_level INTEGER, -- Batería en ese momento
    speed_kmh DECIMAL(5, 2), -- Velocidad en ese momento (si aplica)
    
    -- Metadatos
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    CONSTRAINT fk_location_scooter FOREIGN KEY (scooter_id) REFERENCES scooters(id) ON DELETE CASCADE,
    CONSTRAINT fk_location_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE SET NULL
);

-- Índices para ubicaciones históricas
CREATE INDEX IF NOT EXISTS idx_locations_scooter ON scooter_locations(scooter_id);
CREATE INDEX IF NOT EXISTS idx_locations_trip ON scooter_locations(trip_id) WHERE trip_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_locations_recorded_at ON scooter_locations(recorded_at);
CREATE INDEX IF NOT EXISTS idx_locations_event_type ON scooter_locations(event_type);

-- Tabla de estados del scooter (historial de cambios de estado)
CREATE TABLE IF NOT EXISTS scooter_status_history (
    id SERIAL PRIMARY KEY,
    scooter_id INTEGER NOT NULL,
    old_status VARCHAR(20),
    new_status VARCHAR(20) NOT NULL,
    reason TEXT, -- Razón del cambio (ej: "maintenance", "user started trip", etc.)
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_status_scooter FOREIGN KEY (scooter_id) REFERENCES scooters(id) ON DELETE CASCADE
);

-- Índices para historial de estados
CREATE INDEX IF NOT EXISTS idx_status_history_scooter ON scooter_status_history(scooter_id);
CREATE INDEX IF NOT EXISTS idx_status_history_changed_at ON scooter_status_history(changed_at);

-- Función para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers para actualizar updated_at
CREATE TRIGGER update_scooters_updated_at BEFORE UPDATE ON scooters
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trips_updated_at BEFORE UPDATE ON trips
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

