package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"ewood-backend/internal/websocket"

	gorillaWS "github.com/gorilla/websocket"
)

// generateClientID genera un ID único para un cliente WebSocket
func generateClientID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "client_" + hex.EncodeToString(bytes)
}

var wsHub *websocket.Hub

// InitWebSocketHub inicializa el hub de WebSocket (debe llamarse al inicio)
func InitWebSocketHub() {
	wsHub = websocket.NewHub()
	go wsHub.Run()
	go processIncomingMessages()
	log.Println("✅ WebSocket Hub inicializado")
}

// processIncomingMessages procesa mensajes entrantes del frontend
func processIncomingMessages() {
	if wsHub == nil {
		return
	}
	
	for {
		select {
		case message := <-wsHub.Incoming:
			// Procesar mensaje del frontend
			if err := HandleFrontendMessage(message); err != nil {
				log.Printf("❌ Error procesando mensaje del frontend: %v", err)
			}
		}
	}
}

// GetWebSocketHub retorna el hub de WebSocket
func GetWebSocketHub() *websocket.Hub {
	return wsHub
}

// WebSocketHandler maneja las conexiones WebSocket
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	if wsHub == nil {
		http.Error(w, "WebSocket hub no inicializado", http.StatusInternalServerError)
		return
	}

	// Upgrade la conexión HTTP a WebSocket
	upgrader := gorillaWS.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// En producción, validar el origen
			return true // Permitir todos los orígenes por ahora
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Error haciendo upgrade a WebSocket: %v", err)
		return
	}

	// Generar ID único para el cliente
	clientID := generateClientID()

	// Crear cliente
	client := &websocket.Client{
		Hub:  wsHub,
		Conn: conn,
		Send: make(chan []byte, 256),
		ID:   clientID,
	}

	// Registrar cliente
	wsHub.Register <- client

	// Iniciar goroutines para leer y escribir
	go client.WritePump()
	go client.ReadPump()

	log.Printf("✅ Nuevo cliente WebSocket conectado")
}

