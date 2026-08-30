-- Tokens de password recovery. Uno por request. Expiran a los 30 min.
CREATE TABLE IF NOT EXISTS password_resets (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token VARCHAR(128) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  used_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_resets_token ON password_resets(token) WHERE used_at IS NULL;
CREATE INDEX idx_password_resets_user_id ON password_resets(user_id);
