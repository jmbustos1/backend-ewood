-- Persistencia del consentimiento del usuario a Terminos y Politica de
-- Privacidad. Cada aceptacion queda como una fila (append-only). Al subir
-- la version de un documento, la app re-pide aceptacion y se crea una fila
-- nueva. La ultima fila por (user_id, document) representa el estado actual.
CREATE TABLE IF NOT EXISTS user_consents (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  document VARCHAR(32) NOT NULL, -- 'terms' | 'privacy'
  version VARCHAR(32) NOT NULL,  -- ej. '1.0'
  accepted_at TIMESTAMP NOT NULL DEFAULT NOW(),
  ip_address VARCHAR(64),         -- opcional, para trazabilidad legal
  user_agent VARCHAR(255)         -- opcional
);

CREATE INDEX idx_user_consents_user_id ON user_consents(user_id);
CREATE INDEX idx_user_consents_lookup ON user_consents(user_id, document, accepted_at DESC);
