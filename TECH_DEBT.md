# Tech Debt · backend-ewood

Pendientes técnicos del backend Go. Formato: fecha, dónde, qué, por qué es debt, fix propuesto, prioridad. Más reciente arriba.

---

## 2026-11-04 · Rate limiting en endpoints públicos

- **Dónde**: `internal/router/router.go` — todas las rutas públicas sin middleware de rate limit
- **Qué**: endpoints como `/auth/register`, `/auth/login`, `/auth/forgot-password`, `/auth/reset-password`, `/api/config`, `/api/cities`, `/api/wallet/mp-webhook` aceptan requests sin cuota. Un atacante puede:
  - Fuerza bruta de login (`POST /auth/login`)
  - Enumeración de usuarios via `/auth/forgot-password`
  - Spam del webhook MP (aunque es idempotente, satura logs/DB)
  - DoS trivial sobre `/api/config` o `/api/cities`
- **Por qué es debt**: riesgo de seguridad real. Hoy no hay WAF/CDN adelante, el backend está expuesto directo detrás del reverse proxy.
- **Fix propuesto**:
  - Middleware per-IP con `github.com/ulule/limiter/v3` o `golang.org/x/time/rate`:
    - `/auth/*` → 10 req/min por IP
    - `/api/wallet/mp-webhook` → 60 req/min por IP (whitelist IPs de MP si hay rango estable)
    - `/api/config`, `/api/cities` → 30 req/min por IP
  - Guardar contadores en memoria por ahora (una instancia del backend), después mover a Redis si escala a multi-nodo.
  - Responder `429 Too Many Requests` con `Retry-After`.
- **Prioridad**: alta (bloquea exposición a público sin riesgo)

## 2026-11-04 · Wrap de endpoints públicos con capa de seguridad

- **Dónde**: `internal/router/router.go` — bloque "Rutas públicas" y endpoints admin sin JWT
- **Qué**: varios endpoints públicos deberían tener alguna capa adicional de seguridad más allá del rate limit:
  - `POST /api/admin/scooters` → **crítico**: crea scooters sin ningún auth. Cualquiera puede llenar la DB con scooters ficticios.
  - `GET /env` → expone variables de entorno (aunque enmascara passwords, revela qué infra se usa: hosts, DB name). No debería ser público.
  - `GET /check`, `GET /health` → ok públicos pero conviene limitar body a 200 OK sin filtrar versión de Go, de libs, etc.
  - `POST /api/wallet/mp-webhook` → el handler re-consulta el pago a MP con nuestro Access Token (ok), pero no verifica la firma HMAC `x-signature` que MP envía. Un atacante con el `payment_id` de otro usuario podría disparar webhooks falsos.
  - `POST /api/scooters/{id}/location` → "protegida por device secret en producción" según comentario, verificar que el middleware esté realmente activo.
- **Por qué es debt**: superficie de ataque amplia, especialmente `/api/admin/scooters` sin auth.
- **Fix propuesto** (en orden de prioridad):
  1. `POST /api/admin/scooters`: agregar middleware `RequireAdminKey` que compare header `X-Admin-Key` contra env var `ADMIN_API_KEY`. Si no match → 401.
  2. `GET /env`: mover a `protected` o directamente eliminar — info diagnóstica solo para admins. Alternativa: proteger con `X-Admin-Key`.
  3. `POST /api/wallet/mp-webhook`: validar firma MP (`x-signature` + `x-request-id`) con `MP_WEBHOOK_SECRET`. Doc: https://www.mercadopago.cl/developers/en/docs/your-integrations/notifications/webhooks#bookmark_how_to_validate_the_notification
  4. Verificar que `UpdateScooterLocationHandler` efectivamente valida `X-Device-Secret` en prod (no solo warn log).
- **Prioridad**: alta (especialmente item 1 — `/api/admin/scooters` sin auth)

## 2026-11-04 · Snapshot de tarifa por viaje

- **Dónde**: tabla `trips` (no existe columna `unit_price_cents` ni `tax_rate_bps`)
- **Qué**: `cost_amount` guarda el total cobrado al cerrar el viaje, pero no el precio unitario vigente al momento del viaje. Si la tarifa cambia (ej. hoy pasó de $150 a $120), no hay forma de auditar retroactivamente cuántos trips se cobraron a cada tarifa salvo por fecha.
- **Por qué es debt**: dificulta auditoría legal/contable y análisis de impacto de cambios de precio.
- **Fix propuesto**:
  ```sql
  ALTER TABLE trips
    ADD COLUMN unit_price_cents INT NOT NULL DEFAULT 15000,
    ADD COLUMN tax_rate_bps     INT NOT NULL DEFAULT 1900;
  ```
  Setear ambos al **abrir** el trip (`StartTripHandler`) desde `getPricePerMinute()` y `ivaRate`. Snapshot inmutable.
- **Prioridad**: media

## 2026-11-04 · Fallback hardcoded del precio por minuto

- **Dónde**: `internal/handler/trip.go:38` → `const defaultPricePerMinute = 150.0`
- **Qué**: fallback activo cuando `PRICE_PER_MINUTE` no está seteado. Hoy quedó desactualizado (la tarifa productiva es $120).
- **Por qué es debt**: si alguien levanta un entorno de dev sin env, cobraría una tarifa vieja silenciosamente.
- **Fix propuesto**: cambiar `150.0` → `120.0`, o mejor hacer que `getPricePerMinute()` panic en producción si falta la env var (fail-fast).
- **Prioridad**: baja

## 2026-11-04 · Fallback de MP_BACKEND_BASE_URL con IP vieja

- **Dónde**: `internal/handler/wallet.go:171` → fallback `http://98.92.176.224:8200`
- **Qué**: IP ya no apunta al EC2 actual (ahora `18.214.196.217` y detrás de dominio `api.exwood.dev`).
- **Por qué es debt**: si `MP_BACKEND_BASE_URL` queda vacía, MP envía webhooks a una IP equivocada.
- **Fix propuesto**: cambiar fallback a `https://api.exwood.dev` o eliminar el fallback (panic si falta en prod).
- **Prioridad**: baja (hoy la env var está seteada, no se activa el fallback)
