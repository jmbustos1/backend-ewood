-- Wallet por usuario. Solo una wallet por usuario; balance puede ser negativo
-- (cuando el viaje cuesta mas de lo disponible, el usuario queda en deuda y
-- debe regularizar antes de poder arrendar de nuevo).
CREATE TABLE IF NOT EXISTS wallets (
  user_id    INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  balance    DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_wallets_updated_at
  BEFORE UPDATE ON wallets
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Historial de movimientos de la wallet.
-- type:
--   recharge     = recarga de saldo (MercadoPago / manual)
--   trip_charge  = cobro al finalizar un viaje
--   refund       = reembolso (futuro)
--   adjustment   = ajuste manual de admin (futuro)
CREATE TABLE IF NOT EXISTS wallet_transactions (
  id              SERIAL PRIMARY KEY,
  user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type            VARCHAR(20) NOT NULL,
  amount          DECIMAL(10,2) NOT NULL,        -- positivo en recarga, negativo en trip_charge
  balance_after   DECIMAL(10,2) NOT NULL,        -- saldo final luego de la transaccion
  trip_id         INTEGER REFERENCES trips(id),  -- nullable: solo si type=trip_charge
  mp_payment_id   VARCHAR(100),                  -- nullable: id de MercadoPago para recargas
  description     TEXT,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_wallet_tx_user_id ON wallet_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_trip_id ON wallet_transactions(trip_id);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_created_at ON wallet_transactions(created_at DESC);

-- Crear wallet automaticamente para usuarios existentes
INSERT INTO wallets (user_id, balance)
SELECT id, 0 FROM users
ON CONFLICT (user_id) DO NOTHING;
