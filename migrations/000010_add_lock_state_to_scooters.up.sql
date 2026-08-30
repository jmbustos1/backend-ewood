-- Agrega columnas lock_state y lock_state_updated_at a la tabla scooters.
-- Fuente de verdad: el firmware ESP32 lee el byte Locker del frame SYS STATUS
-- (MODULE=0xFF MSGID=0x06) del hub de sensores y lo reporta en cada telemetria.
-- Backend persiste el valor y lo expone al frontend por WebSocket.
--
-- Valores esperados: 'locked' | 'unlocked' | 'unknown'.
-- 'unknown' es el default para scooters existentes que aun no han reportado.

ALTER TABLE scooters
    ADD COLUMN IF NOT EXISTS lock_state VARCHAR(16) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS lock_state_updated_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_scooters_lock_state ON scooters(lock_state);
