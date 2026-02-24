package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Tiempo de espera para escribir al cliente
	writeWait = 10 * time.Second

	// Tiempo de espera para leer del cliente
	pongWait = 60 * time.Second

	// Frecuencia de ping (debe ser menor que pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Tamaño máximo de mensaje
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// En producción, validar el origen
		return true // Permitir todos los orígenes por ahora
	},
}

// ReadPump lee mensajes del cliente WebSocket
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Error leyendo WebSocket: %v", err)
			}
			break
		}

		// Agregar client_id al mensaje antes de enviarlo al canal
		var msgData map[string]interface{}
		if err := json.Unmarshal(message, &msgData); err == nil {
			// Agregar client_id si no está presente
			if c.ID != "" {
				msgData["_client_id"] = c.ID
			}
			// Re-serializar con client_id
			if modifiedMsg, err := json.Marshal(msgData); err == nil {
				message = modifiedMsg
			}
		}

		// Enviar mensaje al canal de entrada del Hub para procesamiento
		select {
		case c.Hub.Incoming <- message:
			// Mensaje enviado al canal para procesamiento
		default:
			log.Printf("⚠️  Canal de mensajes entrantes lleno, descartando mensaje")
		}
	}
}

// WritePump escribe mensajes al cliente WebSocket
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Enviar mensajes pendientes en el canal
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

