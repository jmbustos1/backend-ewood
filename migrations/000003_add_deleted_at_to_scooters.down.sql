-- Eliminar índices y columnas deleted_at de todas las tablas

-- Scooters
DROP INDEX IF EXISTS idx_scooters_deleted_at;
ALTER TABLE scooters DROP COLUMN IF EXISTS deleted_at;

-- Scooter locations
DROP INDEX IF EXISTS idx_scooter_locations_deleted_at;
ALTER TABLE scooter_locations DROP COLUMN IF EXISTS deleted_at;

-- Trips
DROP INDEX IF EXISTS idx_trips_deleted_at;
ALTER TABLE trips DROP COLUMN IF EXISTS deleted_at;

-- Scooter status history
DROP INDEX IF EXISTS idx_scooter_status_history_deleted_at;
ALTER TABLE scooter_status_history DROP COLUMN IF EXISTS deleted_at;

-- Users
DROP INDEX IF EXISTS idx_users_deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

