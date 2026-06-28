# Identificadores del sistema Ewood

Documento para entender como se identifican y correlacionan las entidades
en el sistema de arriendo de scooters.

---

## Resumen rapido

| ID | Analogia simple | Quien lo genera | Donde vive |
|----|----------------|-----------------|------------|
| scooter_id | "A quien le pregunto" — el scooter fisico | Admin al crear scooter (DB auto-increment) | Base de datos, firmware (hardcoded) |
| client_id | "Quien pregunto" — la sesion del usuario | Backend al conectar WebSocket (random 8 bytes hex) | Memoria del WebSocket Hub |
| request_id | "Que pregunto" — esta peticion especifica | Backend al recibir HTTP o WS (scooter_id + timestamp nano) | Viaja en el comando TCP, vuelve en el ACK |
| message_id | "Numero de ticket" — para la cola interna | Backend (contador atomico incremental por scooter) | Cola de comandos pendientes |
| unlock_code | "Prueba de presencia" — el QR del scooter fisico | Backend al crear scooter (generado automaticamente) | Base de datos + QR pegado en el scooter |
| device_secret | "Credencial del dispositivo" — autenticacion TCP | Variable de entorno (compartida MVP) | Backend env + firmware hardcoded |

---

## Ejemplo completo: Juan arrienda un scooter

### Paso 1 — Juan abre la app

```
Juan abre la app movil
  → App se conecta al WebSocket: ws://servidor/ws/scooters?token=JWT_DE_JUAN
  → Backend valida JWT (middleware RequireAuth)
  → Backend genera client_id = "client_a3f8b2c1" (crypto/rand, 8 bytes hex)
  → Este client_id identifica LA SESION de Juan en el WebSocket
  → Si Juan abre otra pestaña/dispositivo, esa sesion tiene OTRO client_id
```

### Paso 2 — Juan ve scooters en el mapa

```
Backend envia por WebSocket (broadcast a todos los clientes conectados):
  {"type": "location_update", "scooter_id": 1, "data": {"latitude": -36.88, "longitude": -73.04, ...}}

  → Todos los usuarios conectados ven los scooters moverse en el mapa
  → Esto es broadcast: NO usa client_id, va a todos
```

### Paso 3 — Juan escanea el QR del scooter

```
Juan esta frente al scooter fisico
  → Escanea el QR pegado en el scooter
  → El QR contiene el unlock_code, ej: "SCT-1-X7K9"
  → Este codigo prueba que Juan ESTA FISICAMENTE frente al scooter
  → Sin esto, cualquiera podria arrendar desde su casa
```

### Paso 4 — Juan inicia el arriendo

```
App envia: POST /api/trips
  Body: {"scooter_id": 1, "unlock_code": "SCT-1-X7K9"}
  Header: Authorization: Bearer JWT_DE_JUAN

Backend recibe y valida:
  1. JWT valido → extrae user_id de Juan
  2. Scooter 1 existe y status == "available" (con FOR UPDATE para evitar race condition)
  3. unlock_code coincide con scooter.UnlockCode en DB
  4. Juan no tiene otro trip activo
  5. Scooter online (IsScooterOnline)
  6. Bateria suficiente (battery_level > 15%)

Todo OK → Backend genera request_id = "req_1_1711900000000000000"
```

### Paso 5 — Backend envia unlock al scooter

```
Backend encola comando para scooter 1:
  → Genera message_id = 42 (contador atomico, unico por scooter)
  → Inyecta message_id, request_id, client_id, timestamp al JSON
  → Lo mete en la CommandQueue del scooter 1

La goroutine processCommandQueue saca el comando y lo envia por TCP:
  {"command": "unlock", "message_id": 42, "request_id": "req_1_171...",
   "client_id": "client_a3f8b2c1", "timestamp": "2026-03-31T..."}

  → message_id (42) es el "numero de ticket" para trackear este comando
  → request_id es para correlacionar con la peticion HTTP original
  → client_id es para saber a quien mandar el ACK por WebSocket
```

### Paso 6 — Scooter recibe y responde

```
ESP32 recibe el JSON por TCP
  → Parsea el comando "unlock"
  → Verifica request_id no duplicado (dedup cache circular de 20)
  → Envia unlock a actuadores via UART1 (frame binario)
  → Envia ACK inmediato al backend:
    {"type": "ack", "id": "cmd-42", "status": "ok", "command": "unlock",
     "message_id": 42, "request_id": "req_1_171...", "client_id": "client_a3f8b2c1"}
```

### Paso 7 — Backend recibe ACK y responde a Juan

```
Backend recibe ACK por TCP (handleTCPConnection)
  → Detecta type == "ack"
  → Llama manager.ProcessACK(scooterID=1, ackData)
  → Busca message_id=42 en pendingCommands del scooter 1
  → Lo encuentra → cierra canal Done (notifica a quien espera)
  → Ve que client_id = "client_a3f8b2c1"
  → Envia ACK SOLO a Juan por WebSocket (unicast via SendToClient):
    {"type": "command_ack", "scooter_id": 1, "command": "unlock", "status": "success"}

Mientras tanto, SendToScooterAndWaitACK estaba esperando en el canal Done:
  → Se desbloquea → retorna ackReceived=true
  → HTTP handler responde 200 OK a la app de Juan:
    {"success": true, "message": "Trip started", "trip": {...}}
```

### Paso 8 — Juan usa el scooter

```
Scooter envia telemetria cada 5s con datos reales:
  {"scooter_id": 1, "latitude": ..., "battery_level": 50, "lock_state": "unlocked", ...}

  → Backend actualiza DB + broadcast por WebSocket
  → Juan (y todos) ven el scooter moverse en el mapa
```

### Paso 9 — Juan termina el arriendo

```
App envia: PATCH /api/trips/1/end
  Body: {"lat": -36.89, "lng": -73.05}

Backend:
  → Valida que el trip es de Juan y esta activo
  → Envia "lock" al scooter por TCP (misma mecanica: message_id, request_id, ACK)
  → Espera ACK del lock
  → Calcula duracion, distancia
  → Marca trip como completed, scooter como available
  → Responde a Juan con resumen del viaje
```

---

## Diagrama de flujo de IDs

```
                    JUAN (App movil)
                         |
                    [JWT en header]
                         |
                    ┌─────────────┐
                    │  WebSocket  │ ← client_id = "client_a3f8b2c1"
                    │  (sesion)   │    (generado al conectar, random)
                    └──────┬──────┘
                           |
                    POST /api/trips
                    {scooter_id: 1, unlock_code: "SCT-1-X7K9"}
                           |
                    ┌──────┴──────┐
                    │   Backend   │ ← genera request_id = "req_1_171..."
                    │   (Go)      │ ← genera message_id = 42
                    └──────┬──────┘
                           |
                      TCP socket
                      {command: "unlock",
                       message_id: 42,         ← ticket de la cola
                       request_id: "req_1_...",← correlacion HTTP
                       client_id: "client_..."}← para responder a Juan
                           |
                    ┌──────┴──────┐
                    │   ESP32     │ ← scooter_id = 1 (hardcoded en firmware)
                    │  (Scooter)  │ ← device_secret (valida al conectar TCP)
                    └──────┬──────┘
                           |
                      ACK por TCP
                      {type: "ack", message_id: 42, request_id: "req_1_..."}
                           |
                    ┌──────┴──────┐
                    │   Backend   │ ← busca message_id 42 → encuentra client_id
                    └──────┬──────┘
                           |
                    WebSocket unicast a "client_a3f8b2c1" (solo Juan)
                    {type: "command_ack", status: "success"}
```

---

## Notas

- **client_id** es por sesion WebSocket, no por usuario. Si Juan tiene 2 pestanas abiertas,
  cada una tiene su propio client_id. El ACK llega solo a la pestana que inicio la accion.
- **message_id** es un contador atomico POR SCOOTER. Se resetea si el backend se reinicia.
  No es global ni persistente. Solo sirve para la cola en memoria.
- **request_id** incluye timestamp en nanosegundos, lo que lo hace unico en la practica.
  Viaja ida y vuelta (backend → scooter → backend) para cerrar el ciclo.
- **device_secret** en MVP es compartido entre todos los dispositivos. En produccion
  deberia ser unico por scooter.
- **unlock_code** es la prueba de presencia fisica. Se genera al crear el scooter
  y se imprime/pega como QR en el scooter fisico.

---

## Implementacion actual del device_secret (MVP)

### Por que existe
Antes de esta implementacion, cualquier socket TCP al puerto 8201 podia enviar
`{"scooter_id": 1, ...}` y el backend lo registraba como ese scooter sin validar
nada. Cualquiera con la IP del servidor podia suplantar un scooter.

### Como funciona

1. **Backend**: variable de entorno `DEVICE_SECRET`
   - Si esta vacia → no valida (modo dev)
   - Si esta seteada → exige que cada conexion TCP nueva la incluya en su primer mensaje

2. **Firmware**: `#define DEVICE_SECRET "CHANGE_ME_DEVICE_SECRET"` en `app_config.h`
   - El ESP32 inyecta `"device_secret": "..."` en cada JSON de telemetria

3. **Validacion** (en `handler/tcp.go`):
   - Cuando llega el primer mensaje de telemetria de un nuevo `scooter_id`
   - Antes de `RegisterConnection`, compara `tcpReq.DeviceSecret` con `getDeviceSecret()`
   - Si no coincide → log "Conexion rechazada... device_secret invalido o ausente" + `conn.Close()` + `return`
   - Si coincide → registro normal

### Mensaje TCP de telemetria con device_secret

```json
{
  "scooter_id": 1,
  "device_secret": "CHANGE_ME_DEVICE_SECRET",
  "latitude": -36.886767,
  "longitude": -73.044400,
  "battery_level": 50,
  "speed_kmh": 15.3,
  "lock_state": "locked",
  "voltage": 39.11
}
```

### Limitaciones MVP / proximos pasos

- **Misma clave para todos los dispositivos**: si se filtra una, se filtran todas
- **Va en plain text**: TCP sin TLS — quien escuche el trafico puede capturarla
- **Produccion deberia**: clave unica por scooter (guardada en DB), challenge-response
  o TLS mutuo en vez de secret en cada mensaje

### Configuracion

| Donde | Valor |
|-------|-------|
| `backend-ewood/env.production` | `DEVICE_SECRET=CHANGE_ME_DEVICE_SECRET` |
| `esp32ewood/src/app_config.h` | `#define DEVICE_SECRET "CHANGE_ME_DEVICE_SECRET"` |
| Para deshabilitar (dev) | dejar `DEVICE_SECRET` vacio o no setearlo |
