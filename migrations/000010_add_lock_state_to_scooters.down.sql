DROP INDEX IF EXISTS idx_scooters_lock_state;
ALTER TABLE scooters
    DROP COLUMN IF EXISTS lock_state_updated_at,
    DROP COLUMN IF EXISTS lock_state;
