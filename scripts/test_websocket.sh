#!/bin/bash

# Script para probar WebSocket desde la consola
# Requiere: websocat (instalar con: cargo install websocat o usar wscat)

echo "🔌 Conectando a WebSocket: ws://localhost:8200/ws/scooters"
echo ""
echo "Opciones:"
echo "1. Usar websocat (recomendado):"
echo "   websocat ws://localhost:8200/ws/scooters"
echo ""
echo "2. Usar wscat (Node.js):"
echo "   wscat -c ws://localhost:8200/ws/scooters"
echo ""
echo "3. Usar Python (si tienes websockets instalado):"
echo "   python3 scripts/test_websocket.py"
echo ""

# Intentar con websocat si está disponible
if command -v websocat &> /dev/null; then
    echo "✅ Usando websocat..."
    websocat ws://localhost:8200/ws/scooters
elif command -v wscat &> /dev/null; then
    echo "✅ Usando wscat..."
    wscat -c ws://localhost:8200/ws/scooters
else
    echo "❌ No se encontró websocat ni wscat"
    echo ""
    echo "Instalar websocat:"
    echo "  cargo install websocat"
    echo ""
    echo "O instalar wscat:"
    echo "  npm install -g wscat"
    echo ""
    echo "O usar el script Python:"
    echo "  python3 scripts/test_websocket.py"
fi

