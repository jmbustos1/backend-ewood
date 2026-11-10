-- Diagnostico del firmware (bloque "d" de la telemetria TCP).
-- last_diag guarda el ultimo bloque recibido tal cual (JSONB); last_diag_at la
-- hora en que llego. Significado de cada campo: esp32ewood/docs/DIAGNOSTICO.md.

ALTER TABLE scooters
    ADD COLUMN IF NOT EXISTS last_diag JSONB,
    ADD COLUMN IF NOT EXISTS last_diag_at TIMESTAMP;
