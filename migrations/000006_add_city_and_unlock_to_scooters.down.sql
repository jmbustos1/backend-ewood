DROP INDEX IF EXISTS idx_scooters_unlock_code;
DROP INDEX IF EXISTS idx_scooters_city_id;

ALTER TABLE scooters
  DROP COLUMN IF EXISTS unlock_code,
  DROP COLUMN IF EXISTS city_id;
