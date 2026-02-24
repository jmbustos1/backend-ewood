# 🏗️ Arquitectura Recomendada: WebSockets para Scooters en Tiempo Real

## 📋 Resumen

Este documento describe la arquitectura recomendada para manejar actualizaciones de ubicación de scooters en tiempo real usando WebSockets.

---

## 🎯 Flujo de Datos Propuesto

### 1. **Scooters → Backend (HTTP POST)**
```
Scooter envía ubicación:
POST http://<BACKEND_IP>:8200/api/scooters/{id}/location
Body: {
  "latitude": 40.7128,
  "longitude": -74.0060,
  "timestamp": "2024-01-01T12:00:00Z",
  "battery": 85,
  "status": "available"
}
```

### 2. **Backend → Base de Datos**
- Guardar ubicación en la base de datos (PostgreSQL/MySQL/MongoDB)
- Validar y procesar datos

### 3. **Backend → Frontend (WebSocket)**
- Notificar a todos los clientes conectados via WebSocket
- Solo clientes que están viendo el mapa de scooters reciben la actualización

---

## 🔧 Arquitectura Técnica

### Opción 1: WebSocket Único para Todo (Recomendado para empezar)

```
Frontend
  ├─ HTTP GET /api/scooters → Obtener lista inicial
  └─ WebSocket ws://<BACKEND_IP>:8200/ws/scooters
       └─ Subscribe: "scooters:*" (todos) o "scooters:{id}" (específico)
       └─ Recibe updates: { type: "location_update", scooter_id: 123, ... }
```

**Ventajas:**
- Simple de implementar
- Una sola conexión WebSocket
- Fácil de escalar inicialmente

**Desventajas:**
- Todos los clientes reciben todos los updates (puede optimizarse con filtros)

### Opción 2: WebSocket con Rooms/Topics (Recomendado para producción)

```
Frontend
  └─ WebSocket ws://<BACKEND_IP>:8200/ws
       ├─ Join room: "scooters:map" (zona geográfica)
       └─ Recibe solo updates de scooters en esa zona
```

**Ventajas:**
- Más eficiente (solo envía datos relevantes)
- Escalable
- Menos ancho de banda

**Desventajas:**
- Más complejo de implementar

---

## 📦 Stack Tecnológico Recomendado

### Backend (Go)
```go
// Librería recomendada
github.com/gorilla/websocket
```

### Frontend
- **React**: `useWebSocket` hook o librería `socket.io-client`
- **Vanilla JS**: WebSocket API nativo

### Base de Datos
- **PostgreSQL** con extensión PostGIS (para consultas geográficas)
- O **MongoDB** con índices geoespaciales

---

## 🔄 Flujo de Implementación Sugerido

### Fase 1: HTTP Endpoint para Scooters (Ahora)
```
POST /api/scooters/{id}/location
  → Guardar en memoria/cache (map[string]Location)
  → Responder 200 OK
```

### Fase 2: WebSocket Básico
```
GET /ws/scooters
  → Upgrade a WebSocket
  → Enviar updates cuando lleguen ubicaciones
```

### Fase 3: Base de Datos
```
POST /api/scooters/{id}/location
  → Guardar en DB
  → Notificar via WebSocket
  → Responder 200 OK
```

### Fase 4: Optimización con Rooms
```
GET /ws/scooters?bounds={lat1,lng1,lat2,lng2}
  → Solo suscribirse a scooters en ese rango
```

---

## 📝 Estructura de Mensajes WebSocket

### Cliente → Servidor
```json
{
  "action": "subscribe",
  "topic": "scooters:*"
}
```

```json
{
  "action": "subscribe",
  "topic": "scooters:123"
}
```

### Servidor → Cliente
```json
{
  "type": "location_update",
  "scooter_id": 123,
  "data": {
    "latitude": 40.7128,
    "longitude": -74.0060,
    "battery": 85,
    "status": "available",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

---

## 🚀 Recomendación Inicial

**Para empezar rápido:**

1. **Endpoint HTTP simple** para recibir ubicaciones de scooters
2. **WebSocket básico** sin rooms - todos reciben todos los updates
3. **Cache en memoria** (map) antes de agregar base de datos
4. **Base de datos después** una vez que funcione

**Código de ejemplo Go:**

```go
// Estructura para almacenar ubicaciones
type ScooterLocation struct {
    ID        int       `json:"id"`
    Latitude  float64   `json:"latitude"`
    Longitude float64   `json:"longitude"`
    Battery   int       `json:"battery"`
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
}

// Cache en memoria (después migrar a DB)
var scooterLocations = make(map[int]*ScooterLocation)
var wsClients = make(map[*websocket.Conn]bool)
var wsMutex = sync.RWMutex{}
```

---

## 🔐 Seguridad

- Autenticación WebSocket (JWT token en query param)
- Rate limiting para endpoints de scooters
- Validación de coordenadas (lat/lng válidos)
- Validación de timestamps (no futuro, no muy antiguo)

---

## 📊 Escalabilidad Futura

- **Redis Pub/Sub**: Para distribuir updates entre múltiples instancias del backend
- **Load Balancer**: Con sticky sessions para WebSockets
- **Message Queue**: Para procesar updates de scooters de forma asíncrona

---

## ✅ Checklist de Implementación

- [ ] Endpoint HTTP `POST /api/scooters/{id}/location`
- [ ] Estructura de datos para ubicaciones
- [ ] WebSocket endpoint `/ws/scooters`
- [ ] Broadcast de updates a clientes conectados
- [ ] Frontend: Conectar a WebSocket cuando se abre el mapa
- [ ] Frontend: Actualizar marcadores en el mapa en tiempo real
- [ ] Base de datos para persistencia
- [ ] Autenticación y autorización

