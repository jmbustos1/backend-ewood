#!/usr/bin/env python3
"""
Script para simular un scooter enviando actualizaciones de ubicación
Instalar dependencias: pip install requests

Uso:
    python3 scripts/simulate_scooter.py <scooter_id>
    
Ejemplo:
    python3 scripts/simulate_scooter.py 1
"""

import requests
import time
import sys
import json

# Configuración
BASE_URL = "http://localhost:8200"
SCOOTER_ID = None

# Datos estáticos para la simulación
STATIC_LATITUDE = 40.7128   # Nueva York
STATIC_LONGITUDE = -74.0060
STATIC_BATTERY = 85
STATIC_SPEED = 15.5  # km/h

def send_location_update(scooter_id):
    """Envía una actualización de ubicación del scooter"""
    url = f"{BASE_URL}/api/scooters/{scooter_id}/location"
    
    data = {
        "latitude": STATIC_LATITUDE,
        "longitude": STATIC_LONGITUDE,
        "battery_level": STATIC_BATTERY,
        "speed_kmh": STATIC_SPEED
    }
    
    try:
        response = requests.post(url, json=data, timeout=5)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Actualización enviada - Scooter {scooter_id}: {result.get('message', 'OK')}")
            return True
        else:
            print(f"❌ Error {response.status_code}: {response.text}")
            return False
    except requests.exceptions.ConnectionError:
        print(f"❌ Error: No se pudo conectar al servidor en {BASE_URL}")
        print("   Asegúrate de que el servidor esté corriendo")
        return False
    except Exception as e:
        print(f"❌ Error: {e}")
        return False

def main():
    global SCOOTER_ID
    
    # Obtener scooter_id de argumentos
    if len(sys.argv) > 1:
        try:
            SCOOTER_ID = int(sys.argv[1])
        except ValueError:
            print("❌ Error: El scooter_id debe ser un número")
            print(f"   Uso: {sys.argv[0]} <scooter_id>")
            sys.exit(1)
    else:
        print("❌ Error: Se requiere el ID del scooter")
        print(f"   Uso: {sys.argv[0]} <scooter_id>")
        print(f"   Ejemplo: {sys.argv[0]} 1")
        sys.exit(1)
    
    print(f"🛴 Simulando scooter ID: {SCOOTER_ID}")
    print(f"📍 Ubicación estática: {STATIC_LATITUDE}, {STATIC_LONGITUDE}")
    print(f"🔋 Batería: {STATIC_BATTERY}%")
    print(f"⚡ Velocidad: {STATIC_SPEED} km/h")
    print(f"🔄 Enviando actualizaciones cada 3 segundos...")
    print(f"💡 Presiona Ctrl+C para detener\n")
    
    update_count = 0
    
    try:
        while True:
            update_count += 1
            print(f"[{update_count}] ", end="", flush=True)
            send_location_update(SCOOTER_ID)
            time.sleep(3)
    except KeyboardInterrupt:
        print(f"\n\n👋 Simulación detenida. Total de actualizaciones enviadas: {update_count}")

if __name__ == "__main__":
    main()

