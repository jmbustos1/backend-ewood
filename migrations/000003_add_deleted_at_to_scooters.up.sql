-- Agregar columna deleted_at para soft deletes en todas las tablas que lo necesitan

-- Scooters
ALTER TABLE scooters ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_scooters_deleted_at ON scooters(deleted_at);

-- Scooter locations
ALTER TABLE scooter_locations ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_scooter_locations_deleted_at ON scooter_locations(deleted_at);

-- Trips
ALTER TABLE trips ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_trips_deleted_at ON trips(deleted_at);

-- Scooter status history
ALTER TABLE scooter_status_history ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_scooter_status_history_deleted_at ON scooter_status_history(deleted_at);

-- Users (si no existe ya)
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

