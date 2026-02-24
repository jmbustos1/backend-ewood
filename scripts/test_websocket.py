#!/usr/bin/env python3
"""
Script para probar WebSocket desde la consola
Instalar dependencias: pip install websockets
"""

import asyncio
import json
import websockets
import sys

async def test_websocket():
    uri = "ws://localhost:8200/ws/scooters"
    
    print(f"🔌 Conectando a {uri}...")
    
    try:
        async with websockets.connect(uri) as websocket:
            print("✅ Conectado al WebSocket!")
            print("📡 Esperando mensajes...")
            print("💡 Presiona Ctrl+C para salir\n")
            
            # Enviar mensaje de prueba (opcional)
            # await websocket.send(json.dumps({"action": "subscribe", "topic": "scooters:*"}))
            
            # Escuchar mensajes
            try:
                async for message in websocket:
                    try:
                        data = json.loads(message)
                        print(f"📨 Mensaje recibido:")
                        print(json.dumps(data, indent=2))
                        print("-" * 50)
                    except json.JSONDecodeError:
                        print(f"📨 Mensaje recibido (texto): {message}")
                        print("-" * 50)
            except KeyboardInterrupt:
                print("\n👋 Desconectando...")
                
    except websockets.exceptions.ConnectionRefused:
        print("❌ Error: No se pudo conectar al servidor")
        print("   Asegúrate de que el servidor esté corriendo en localhost:8200")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    try:
        asyncio.run(test_websocket())
    except KeyboardInterrupt:
        print("\n👋 Saliendo...")

