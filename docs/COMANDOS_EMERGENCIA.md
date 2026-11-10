# Comandos de emergencia a scooters

Endpoint de operación para enviar un comando a un scooter **fuera del flujo de viajes**, por ejemplo para bloquear uno que quedó abierto. No lo usa la app: es solo para el equipo.

## Canales de comando hacia un scooter

| Canal | Quién | Comandos | Efecto en la base |
|---|---|---|---|
| `POST /api/trips` | Usuario con sesión | `unlock` | Crea el viaje y marca el scooter `in_use` |
| `PATCH /api/trips/{id}/end` | Dueño del viaje | `lock` | Cierra y cobra el viaje, scooter `available` |
| `POST /api/admin/scooters/{id}/command` | Equipo, con `X-Admin-Key` | Lista cerrada (abajo) | **Ninguno**: solo envía el comando |

No existen otros canales. Desde el 2026-11-09:

- `POST /api/scooters/{id}/unlock` y `/lock` ya no existen.
- El WebSocket `/ws/scooters` es solo de lectura (mapa en vivo). Si un cliente envía `send_to_scooter`, se ignora y queda en el log como `🚫 WebSocket: send_to_scooter ignorado`.

## Uso

```bash
curl -X POST https://api.exwood.dev/api/admin/scooters/4/command \
  -H "X-Admin-Key: $ADMIN_COMMAND_KEY" \
  -H "Content-Type: application/json" \
  -d '{"command":"lock"}'
```

Respuestas:

| HTTP | Cuerpo | Significado |
|---|---|---|
| 200 | `{"ack":true, ...}` | El scooter confirmó el comando |
| 504 | `{"ack":false, ...}` | El scooter no confirmó en 25 s |
| 503 | `scooter_not_connected` | El scooter no está conectado al backend |
| 503 | `admin_commands_disabled` | `ADMIN_COMMAND_KEY` no está configurada o tiene menos de 32 caracteres |
| 401 | `unauthorized` | Clave ausente o incorrecta (queda en el log con la IP de origen) |
| 400 | `command_not_allowed` | El comando no está en la lista |

Cada uso queda en el log del backend:

```
🛠️  Comando admin "lock" a scooter 4 (origen <ip>): ack=true message_id=12 err=<nil>
```

## Comandos permitidos

| Comando | Qué hace | Requiere firmware |
|---|---|---|
| `lock` | Bloquea el actuador | Actual |
| `unlock` | Desbloquea el actuador | Actual |

Planificados para el próximo flasheo: `gps_status`, `gps_recycle`, `gps_cold` y `reboot`.

### Reglas para agregar comandos

1. **Nunca AT libres.** Solo comandos con nombre, implementados en el firmware. Un AT mal puesto puede cambiar el APN o dejar el módem inaccesible.
2. **El nombre no puede contener `lock`.** El firmware tiene una ruta de texto plano que busca esa palabra dentro del mensaje (`sim7600.c`, ruta texto plano).
3. Agregarlo en `adminCommands` (`internal/handler/admin_command.go`) **después** de que el firmware lo soporte, y documentarlo en esta tabla.

## Notas

- El comando admin no crea ni cierra viajes. Un `unlock` admin deja el scooter desbloqueado con estado `available`; hay que bloquearlo al terminar.
- La clave `ADMIN_COMMAND_KEY` vive solo en `env.production` y en el gestor de contraseñas del equipo. No va en la app, en el repo ni en la documentación.
- Código: `internal/handler/admin_command.go`. Tests: `admin_command_test.go` y `internal/router/router_test.go`.
