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
| 200 | `{"ack":true, "status":"ok", "result":{...}}` | El scooter confirmó el comando. `result` solo en comandos que devuelven datos |
| 502 | `{"ack":true, "status":"error", "result":{"error":...}}` | El firmware respondió, pero el comando falló (ej. `gps_busy`, `unknown_command`) |
| 504 | `{"ack":false, ...}` | El scooter no confirmó en 25 s. Con firmware anterior a los comandos GPS, un comando GPS termina así |
| 409 | `scooter_in_trip` | `reboot` o `gps_cold` con el scooter en un viaje activo. Repetir con `"force": true` si es necesario |
| 404 | `scooter_not_found` | El scooter no existe (solo en comandos que verifican viajes) |
| 503 | `scooter_not_connected` | El scooter no está conectado al backend |
| 503 | `admin_commands_disabled` | `ADMIN_COMMAND_KEY` no está configurada o tiene menos de 32 caracteres |
| 401 | `unauthorized` | Clave ausente o incorrecta (queda en el log con la IP de origen) |
| 400 | `command_not_allowed` | El comando no está en la lista |

Cada uso queda en el log del backend:

```
🛠️  Comando admin "lock" a scooter 4 (origen <ip>): ack=true message_id=12 err=<nil>
```

## Comandos permitidos

| Comando | Qué hace en el firmware | `result` | En viaje |
|---|---|---|---|
| `lock` | Bloquea el actuador | — | Permitido |
| `unlock` | Desbloquea el actuador | — | Permitido |
| `gps_status` | `AT+CGPS?` y `AT+CGNSSINFO` | Ver abajo | Permitido |
| `gps_recycle` | Encola: `CGPS=0`, espera hasta que `AT+CGPS?` confirme apagado (máx. 30 s), `CGPS=1,1` | `{"queued":true}` o error `gps_busy` | Permitido |
| `gps_cold` | Igual que `gps_recycle`, más `AT+CGPSDEL` (borra efemérides: arranque en frío, TTFF más largo) | `{"queued":true}` o error `gps_busy` | Requiere `force` |
| `reboot` | Envía el ACK, espera 1,5 s y `esp_restart()` | — | Requiere `force` |

`lock`/`unlock` funcionan con cualquier firmware. Los comandos GPS y `reboot` requieren el firmware con comandos de mantenimiento (rama `chore/consolidate-fixes-aug`, desde `64d7143`; verificar con `d.fw`). Con un firmware anterior no hay ACK y el endpoint responde 504.

`gps_recycle` y `gps_cold` son **asíncronos**: el ACK confirma que quedó encolado, no el resultado. Para ver el efecto, usar `gps_status` unos segundos después o mirar el diagnóstico (`d.gr` sube al terminar; `d.gv`/`d.fa` muestran el fix). Solo corre una acción a la vez.

Después de `reboot`, el scooter se desconecta y se reconecta solo en unos segundos; su diagnóstico muestra `up` bajo y `rr=3` (reinicio por software).

### `result` de `gps_status`

```json
{"on":1,"mode":1,"fix":3,"sv":[8,4,2],"fa":3,"gr":0,"busy":0}
```

| Campo | Significado |
|---|---|
| `on` | 1 = motor GNSS encendido (`AT+CGPS?`), 0 = apagado, -1 = sin respuesta |
| `mode` | 1 = standalone |
| `fix` | 2 = 2D, 3 = 3D, -1 = sin fix (`AT+CGNSSINFO`) |
| `sv` | Satélites en uso: `[GPS, GLONASS, BEIDOU]`; -1 = sin dato |
| `fa` | Segundos desde el último fix real; -1 = nunca desde el arranque |
| `gr` | Reciclajes/arranques en frío desde el arranque (automáticos y por comando) |
| `busy` | 1 = hay un reciclaje o arranque en frío en curso |

Lectura rápida: `on=0` → el motor está apagado (usar `gps_recycle`). `on=1` y `sv` en 0 por varios minutos → no ve satélites: bajo techo o antena. `on=1`, `sv` > 0 y `fix=-1` → señal débil, esperar o `gps_cold`.

Sintaxis de los AT según el manual SIMCom SIM7500/SIM7600 AT Command Manual V3.00 (págs. 369–388); copia local del equipo, no se distribuye en el repo.

### Reglas para agregar comandos

1. **Nunca AT libres.** Solo comandos con nombre, implementados en el firmware. Un AT mal puesto puede cambiar el APN o dejar el módem inaccesible.
2. **El nombre no puede contener `lock`.** El firmware tiene una ruta de texto plano que busca esa palabra dentro del mensaje (`sim7600.c`, ruta texto plano).
3. Implementarlo en `process_command` (`esp32ewood/src/sim7600.c`). Un comando desconocido responde ACK con `status: "error"`.
4. Agregarlo en `adminCommands` (`internal/handler/admin_command.go`) **después** de que el firmware lo soporte; si interrumpe al scooter, también en `disruptiveCommands`. Documentarlo en esta tabla.

## Notas

- El comando admin no crea ni cierra viajes. Un `unlock` admin deja el scooter desbloqueado con estado `available`; hay que bloquearlo al terminar.
- La clave `ADMIN_COMMAND_KEY` vive solo en `env.production` y en el gestor de contraseñas del equipo. No va en la app, en el repo ni en la documentación.
- Código: `internal/handler/admin_command.go`. Tests: `admin_command_test.go` y `internal/router/router_test.go`.
