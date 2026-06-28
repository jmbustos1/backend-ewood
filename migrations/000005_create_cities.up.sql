CREATE TABLE IF NOT EXISTS cities (
  id         SERIAL PRIMARY KEY,
  name       VARCHAR(100) NOT NULL,
  country    VARCHAR(100) NOT NULL DEFAULT 'Chile',
  lat        DECIMAL(10,8) NOT NULL,
  lng        DECIMAL(11,8) NOT NULL,
  radius_km  DECIMAL(5,2) NOT NULL DEFAULT 5.0,
  active     BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cities_active ON cities(active);

CREATE TRIGGER update_cities_updated_at
  BEFORE UPDATE ON cities
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Seed: primera ciudad
INSERT INTO cities (name, country, lat, lng, radius_km)
VALUES ('Concepción', 'Chile', -36.8270, -73.0503, 10.0);
