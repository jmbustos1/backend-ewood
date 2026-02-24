-- Rollback: Eliminar esquema de scooters

-- Eliminar triggers
DROP TRIGGER IF EXISTS update_trips_updated_at ON trips;
DROP TRIGGER IF EXISTS update_scooters_updated_at ON scooters;

-- Eliminar función
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Eliminar tablas (en orden inverso por dependencias)
DROP TABLE IF EXISTS scooter_status_history;
DROP TABLE IF EXISTS scooter_locations;
DROP TABLE IF EXISTS trips;
DROP TABLE IF EXISTS scooters;

