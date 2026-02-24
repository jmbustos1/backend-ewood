#!/usr/bin/env python3
"""
Script para simular un scooter enviando actualizaciones de ubicación por TCP
No requiere dependencias externas (solo librerías estándar de Python)

Uso:
    python3 scripts/simulate_scooter_tcp.py <scooter_id>
    
Ejemplo:
    python3 scripts/simulate_scooter_tcp.py 1

Variables de entorno:
    TCP_HOST: Host del servidor TCP (por defecto: localhost)
    TCP_PORT: Puerto del servidor TCP (por defecto: 8201)
"""

import socket
import time
import sys
import json
import os

# Configuración desde variables de entorno
TCP_HOST = os.getenv("TCP_HOST", "localhost")
TCP_PORT = int(os.getenv("TCP_PORT", "8201"))
SCOOTER_ID = None

# Datos estáticos para la simulación
STATIC_LATITUDE = 40.7128   # Nueva York
STATIC_LONGITUDE = -74.0060
STATIC_BATTERY = 85
STATIC_SPEED = 15.5  # km/h

def send_location_update_tcp(scooter_id, host, port):
    """Envía una actualización de ubicación del scooter por TCP"""
    try:
        # Crear socket TCP
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)  # Timeout de 5 segundos
        
        # Conectar al servidor
        sock.connect((host, port))
        
        # Preparar datos JSON
        data = {
            "scooter_id": scooter_id,
            "latitude": STATIC_LATITUDE,
            "longitude": STATIC_LONGITUDE,
            "battery_level": STATIC_BATTERY,
            "speed_kmh": STATIC_SPEED
        }
        
        # Convertir a JSON y agregar salto de línea
        json_data = json.dumps(data) + "\n"
        
        # Enviar datos
        sock.sendall(json_data.encode('utf-8'))
        
        # Cerrar conexión
        sock.close()
        
        print(f"✅ Actualización enviada - Scooter {scooter_id}")
        return True
        
    except socket.gaierror:
        print(f"❌ Error: No se pudo resolver el host '{host}'")
        return False
    except socket.timeout:
        print(f"❌ Error: Timeout al conectar con {host}:{port}")
        return False
    except ConnectionRefusedError:
        print(f"❌ Error: Conexión rechazada en {host}:{port}")
        print("   Asegúrate de que el servidor TCP esté corriendo")
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
    print(f"🌐 Conectando a TCP {TCP_HOST}:{TCP_PORT}")
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
            send_location_update_tcp(SCOOTER_ID, TCP_HOST, TCP_PORT)
            time.sleep(3)
    except KeyboardInterrupt:
        print(f"\n\n👋 Simulación detenida. Total de actualizaciones enviadas: {update_count}")

if __name__ == "__main__":
    main()
