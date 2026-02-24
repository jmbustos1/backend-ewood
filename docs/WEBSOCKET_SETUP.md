# 🔌 WebSocket Setup - Scooters en Tiempo Real

## 📋 Resumen

Este documento explica cómo usar el sistema de WebSocket para recibir actualizaciones de ubicación de scooters en tiempo real.

---

## 🚀 Endpoints Disponibles

### WebSocket
- **URL**: `ws://localhost:8200/ws/scooters`
- **Descripción**: Conexión WebSocket para recibir actualizaciones de scooters

### API REST

#### Crear Scooter (Admin)
```bash
POST /api/admin/scooters
Headers:
  X-Admin-Key: <SCOOTER_ADMIN_KEY>
Body:
{
  "code": "SCO-001",
  "model": "Modelo X",
  "auth_key": "clave-secreta-del-scooter"
}
```

#### Actualizar Ubicación de Scooter
```bash
POST /api/scooters/{id}/location
Body:
{
  "latitude": 40.7128,
  "longitude": -74.0060,
  "battery_level": 85,
  "speed_kmh": 15.5
}
```

#### Listar Scooters
```bash
GET /api/scooters
```

#### Obtener Scooter por ID
```bash
GET /api/scooters/{id}
```

---

## 🔐 Configuración de Seguridad

### Variable de Entorno Requerida

Para crear scooters, necesitas configurar la clave de administración:

```bash
export SCOOTER_ADMIN_KEY="tu-clave-secreta-aqui"
```

Agrega esto a tu `docker-compose-local.yml`:

```yaml
environment:
  - SCOOTER_ADMIN_KEY=tu-clave-secreta-aqui
```

---

## 🧪 Probar WebSocket

### Opción 1: Script Python (Recomendado)

```bash
# Instalar dependencias
pip install websockets

# Ejecutar script
python3 scripts/test_websocket.py
```

### Opción 2: websocat

```bash
# Instalar websocat
cargo install websocat

# Conectar
websocat ws://localhost:8200/ws/scooters
```

### Opción 3: wscat (Node.js)

```bash
# Instalar wscat
npm install -g wscat

# Conectar
wscat -c ws://localhost:8200/ws/scooters
```

### Opción 4: Script Bash

```bash
chmod +x scripts/test_websocket.sh
./scripts/test_websocket.sh
```

---

## 🛴 Simular Scooter

### Script Python

```bash
# Instalar dependencias
pip install requests

# Ejecutar simulación (requiere scooter_id)
python3 scripts/simulate_scooter.py 1
```

Este script enviará actualizaciones de ubicación cada 3 segundos con datos estáticos.

---

## 📨 Formato de Mensajes WebSocket

### Mensaje Recibido (Servidor → Cliente)

```json
{
  "type": "location_update",
  "scooter_id": 1,
  "data": {
    "id": 1,
    "code": "SCO-001",
    "latitude": 40.7128,
    "longitude": -74.0060,
    "battery_level": 85,
    "status": "available",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

---

## 🔄 Flujo Completo de Prueba

### 1. Iniciar el servidor

```bash
make dockerup
```

### 2. Crear un scooter (en otra terminal)

```bash
curl -X POST http://localhost:8200/api/admin/scooters \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: tu-clave-secreta-aqui" \
  -d '{
    "code": "SCO-001",
    "model": "Test Model",
    "auth_key": "test-key"
  }'
```

### 3. Conectar al WebSocket (en otra terminal)

```bash
python3 scripts/test_websocket.py
```

### 4. Simular actualizaciones (en otra terminal)

```bash
python3 scripts/simulate_scooter.py 1
```

### 5. Ver actualizaciones en tiempo real

Deberías ver mensajes JSON apareciendo en la terminal del WebSocket cada 3 segundos.

---

## 🐛 Troubleshooting

### Error: "WebSocket hub no inicializado"
- Asegúrate de que el servidor esté corriendo
- Verifica que `handler.InitWebSocketHub()` se llame en `main.go`

### Error: "Unauthorized: Invalid admin key"
- Verifica que `SCOOTER_ADMIN_KEY` esté configurada
- Usa el mismo valor en el header `X-Admin-Key`

### No se reciben mensajes en WebSocket
- Verifica que el scooter exista en la base de datos
- Asegúrate de que las actualizaciones se estén enviando correctamente
- Revisa los logs del servidor para errores

---

## 📝 Notas

- Los mensajes WebSocket se envían a **todos** los clientes conectados
- En producción, considera implementar filtros por zona geográfica
- La autenticación del scooter (auth_key) aún no está implementada en el endpoint de actualización

