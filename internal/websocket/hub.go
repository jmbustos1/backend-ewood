package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub mantiene el conjunto de clientes WebSocket activos y los mensajes de broadcast
type Hub struct {
	// Clientes conectados
	clients map[*Client]bool

	// Mapa de clientes por ID para correlación
	clientsByID map[string]*Client // client_id -> cliente

	// Canal para registrar nuevos clientes
	Register chan *Client

	// Canal para desregistrar clientes
	Unregister chan *Client

	// Canal para broadcast de mensajes a todos los clientes
	Broadcast chan []byte

	// Canal para mensajes entrantes del frontend (para procesar comandos)
	Incoming chan []byte

	// Mutex para acceso concurrente seguro
	mu sync.RWMutex
}

// Client representa un cliente WebSocket
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	ID     string // ID único del cliente para correlación
	mu     sync.Mutex
}

// NewHub crea un nuevo Hub
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		clientsByID: make(map[string]*Client),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Broadcast:   make(chan []byte, 256),
		Incoming:    make(chan []byte, 256),
	}
}

// Run inicia el hub (debe ejecutarse en una goroutine)
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			if client.ID != "" {
				h.clientsByID[client.ID] = client
			}
			h.mu.Unlock()
			log.Printf("✅ Cliente WebSocket conectado (ID: %s). Total: %d", client.ID, len(h.clients))

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if client.ID != "" {
					delete(h.clientsByID, client.ID)
				}
				close(client.Send)
				log.Printf("❌ Cliente WebSocket desconectado (ID: %s). Total: %d", client.ID, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastMessage envía un mensaje a todos los clientes conectados
func (h *Hub) BroadcastMessage(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case h.Broadcast <- data:
	default:
		log.Println("⚠️  Canal de broadcast lleno, descartando mensaje")
	}

	return nil
}

// GetClientCount retorna el número de clientes conectados
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientByID retorna un cliente por su ID
func (h *Hub) GetClientByID(clientID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clientsByID[clientID]
}

// SendToClient envía un mensaje a un cliente específico por ID
func (h *Hub) SendToClient(clientID string, message interface{}) error {
	client := h.GetClientByID(clientID)
	if client == nil {
		return fmt.Errorf("cliente con ID %s no encontrado", clientID)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case client.Send <- data:
		return nil
	default:
		return fmt.Errorf("canal del cliente %s lleno", clientID)
	}
}

