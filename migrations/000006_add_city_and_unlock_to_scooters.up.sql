ALTER TABLE scooters
  ADD COLUMN IF NOT EXISTS unlock_code VARCHAR(50),
  ADD COLUMN IF NOT EXISTS city_id     INTEGER REFERENCES cities(id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scooters_unlock_code ON scooters(unlock_code) WHERE unlock_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scooters_city_id ON scooters(city_id);
