package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCPConnection representa una conexión TCP activa de un scooter
type TCPConnection struct {
	Conn       net.Conn
	ScooterID  uint
	LastUpdate time.Time
	RemoteAddr string
	mu         sync.Mutex
}

// PendingCommand representa un comando pendiente de ACK
type PendingCommand struct {
	MessageID         uint64
	Command           []byte
	ScooterID         uint
	SentAt            time.Time
	RetryCount        int
	OriginalTimestamp string
	CommandType       string
	ClientID          string // ID del cliente WebSocket (opcional)
	RequestID         string // ID único de la petición (opcional)
	Done              chan struct{} // se cierra cuando llega ACK o falla (para espera asíncrona)
}

// CommandQueue gestiona la cola de comandos para un scooter
type CommandQueue struct {
	queue          chan *PendingCommand
	pendingCommands map[uint64]*PendingCommand // message_id -> comando
	mu             sync.RWMutex
	messageIDCounter uint64
}

// TCPConnectionManager gestiona las conexiones TCP activas de los scooters
type TCPConnectionManager struct {
	mu                sync.RWMutex
	activeConnections map[uint]*TCPConnection // scooter_id -> conexión
	commandQueues     map[uint]*CommandQueue  // scooter_id -> cola de comandos
	inactivityTimeout time.Duration
	ackTimeout        time.Duration
	maxRetries        int
}

var tcpManager *TCPConnectionManager

// InitTCPManager inicializa el gestor de conexiones TCP
func InitTCPManager() {
	tcpManager = &TCPConnectionManager{
		activeConnections: make(map[uint]*TCPConnection),
		commandQueues:     make(map[uint]*CommandQueue),
		inactivityTimeout: 60 * time.Second, // 60 segundos de timeout
		ackTimeout:        30 * time.Second, // 30 segundos para esperar ACK (aumentado para IoT lento)
		maxRetries:        3,                 // Máximo 3 reintentos
	}
	
	// Iniciar goroutine para limpiar conexiones inactivas
	go tcpManager.cleanupInactiveConnections()
	
	// Iniciar goroutine para limpiar comandos pendientes expirados
	go tcpManager.cleanupExpiredCommands()
	
	log.Println("✅ TCP Connection Manager inicializado con sistema de cola y ACK")
}

// GetTCPManager retorna el gestor de conexiones TCP
func GetTCPManager() *TCPConnectionManager {
	return tcpManager
}

// RegisterConnection registra una nueva conexión TCP para un scooter
func (m *TCPConnectionManager) RegisterConnection(scooterID uint, conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	remoteAddr := conn.RemoteAddr().String()
	
	// Si ya existe una conexión para este scooter, cerrarla primero
	if oldConn, exists := m.activeConnections[scooterID]; exists {
		log.Printf("🔄 Reemplazando conexión existente para scooter %d (anterior: %s)", scooterID, oldConn.RemoteAddr)
		oldConn.mu.Lock()
		oldConn.Conn.Close()
		oldConn.mu.Unlock()
	}

	// Registrar nueva conexión
	m.activeConnections[scooterID] = &TCPConnection{
		Conn:       conn,
		ScooterID:  scooterID,
		LastUpdate: time.Now(),
		RemoteAddr: remoteAddr,
	}

	// Crear cola de comandos si no existe
	if _, exists := m.commandQueues[scooterID]; !exists {
		m.commandQueues[scooterID] = &CommandQueue{
			queue:           make(chan *PendingCommand, 100), // Buffer de 100 comandos
			pendingCommands: make(map[uint64]*PendingCommand),
		}
		// Iniciar goroutine para procesar la cola
		go m.processCommandQueue(scooterID)
	}

	log.Printf("✅ Scooter %d registrado (conexión desde %s). Total activos: %d", scooterID, remoteAddr, len(m.activeConnections))

	// Notificar al frontend que el scooter está online
	m.notifyScooterStatus(scooterID, "online")
}

// UpdateLastActivity actualiza el timestamp de última actividad
func (m *TCPConnectionManager) UpdateLastActivity(scooterID uint) {
	m.mu.RLock()
	conn, exists := m.activeConnections[scooterID]
	m.mu.RUnlock()

	if exists {
		conn.mu.Lock()
		conn.LastUpdate = time.Now()
		conn.mu.Unlock()
	}
}

// UnregisterConnection elimina una conexión del mapa
func (m *TCPConnectionManager) UnregisterConnection(scooterID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.activeConnections[scooterID]; exists {
		delete(m.activeConnections, scooterID)
		log.Printf("❌ Scooter %d desconectado. Total activos: %d", scooterID, len(m.activeConnections))
		
		// Cerrar cola de comandos si existe
		if queue, queueExists := m.commandQueues[scooterID]; queueExists {
			close(queue.queue)
			delete(m.commandQueues, scooterID)
		}
		
		// Notificar al frontend que el scooter está offline
		m.notifyScooterStatus(scooterID, "offline")
		
		// Cerrar conexión
		conn.mu.Lock()
		conn.Conn.Close()
		conn.mu.Unlock()
	}
}

// ProcessACK procesa un ACK recibido del IoT
func (m *TCPConnectionManager) ProcessACK(scooterID uint, ackData map[string]interface{}) {
	m.mu.RLock()
	queue, exists := m.commandQueues[scooterID]
	m.mu.RUnlock()

	if !exists || queue == nil {
		return
	}

	// Extraer message_id o request_id del ACK
	var messageID uint64
	var requestID string
	
	// Intentar obtener message_id directamente
	if mid, ok := ackData["message_id"].(float64); ok {
		messageID = uint64(mid)
	} else {
		// Si no hay message_id, buscar por request_id
		if reqID, ok := ackData["request_id"].(string); ok {
			requestID = reqID
			// Buscar comando por request_id
			queue.mu.RLock()
			for mid, cmd := range queue.pendingCommands {
				if cmd.RequestID == requestID {
					messageID = mid
					queue.mu.RUnlock()
					break
				}
			}
			if messageID == 0 {
				queue.mu.RUnlock()
			}
		}
		
		if messageID == 0 {
			log.Printf("⚠️  ACK recibido sin message_id ni request_id válido: %v", ackData)
			return
		}
	}

	// Buscar y remover comando pendiente
	queue.mu.Lock()
	cmd, cmdExists := queue.pendingCommands[messageID]
	if cmdExists {
		delete(queue.pendingCommands, messageID)
	}
	queue.mu.Unlock()

	if cmdExists && cmd.Done != nil {
		close(cmd.Done) // notificar a quien espera (SendToScooterAndWaitACK)
	}

	if !cmdExists {
		log.Printf("⚠️  ACK recibido para comando desconocido (scooter %d, message_id: %d, request_id: %s)", 
			scooterID, messageID, requestID)
		return
	}

	log.Printf("✅ ACK recibido para comando (scooter %d, message_id: %d, tipo: %s)", 
		scooterID, messageID, cmd.CommandType)

	// Notificar al cliente WebSocket si hay client_id
	if cmd.ClientID != "" {
		if hub := GetWebSocketHub(); hub != nil {
			ackMessage := map[string]interface{}{
				"type":       "command_ack",
				"scooter_id": scooterID,
				"command":    cmd.CommandType,
				"message_id": messageID,
				"request_id": cmd.RequestID,
				"status":     "success",
				"timestamp":  time.Now().Format(time.RFC3339),
			}

			if err := hub.SendToClient(cmd.ClientID, ackMessage); err != nil {
				log.Printf("⚠️  Error enviando ACK al cliente %s: %v", cmd.ClientID, err)
			} else {
				log.Printf("📤 ACK enviado al cliente WebSocket %s", cmd.ClientID)
			}
		}
	}
}

// cleanupExpiredCommands limpia comandos pendientes que expiraron
func (m *TCPConnectionManager) cleanupExpiredCommands() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		queues := make(map[uint]*CommandQueue)
		for k, v := range m.commandQueues {
			queues[k] = v
		}
		m.mu.RUnlock()

		now := time.Now()
		for scooterID, queue := range queues {
			queue.mu.Lock()
			var expiredIDs []uint64
			var expiredCmds []*PendingCommand
			for messageID, cmd := range queue.pendingCommands {
				if now.Sub(cmd.SentAt) > (m.ackTimeout * 2) {
					expiredIDs = append(expiredIDs, messageID)
					expiredCmds = append(expiredCmds, cmd)
				}
			}
			for _, messageID := range expiredIDs {
				delete(queue.pendingCommands, messageID)
				log.Printf("⏱️  Comando expirado removido (scooter %d, message_id: %d)", scooterID, messageID)
			}
			queue.mu.Unlock()
			for _, cmd := range expiredCmds {
				if cmd.Done != nil {
					close(cmd.Done)
				}
			}
		}
	}
}

// SendToScooter envía un mensaje a un scooter específico usando la cola.
// Retorna el message_id asignado (para correlación/espera de ACK).
// clientID y requestID son opcionales para correlación con clientes WebSocket.
func (m *TCPConnectionManager) SendToScooter(scooterID uint, message []byte, clientID, requestID string) (uint64, error) {
	m.mu.RLock()
	_, exists := m.activeConnections[scooterID]
	queue, queueExists := m.commandQueues[scooterID]
	m.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("scooter %d no está conectado", scooterID)
	}

	if !queueExists {
		return 0, fmt.Errorf("cola de comandos no inicializada para scooter %d", scooterID)
	}

	// Parsear el mensaje para extraer información del comando
	var cmdData map[string]interface{}
	commandType := "unknown"
	if err := json.Unmarshal(message, &cmdData); err == nil {
		if cmd, ok := cmdData["command"].(string); ok {
			commandType = cmd
		}
	}

	// Generar message_id único
	messageID := atomic.AddUint64(&queue.messageIDCounter, 1)
	
	// Agregar message_id y timestamp al mensaje
	if cmdData == nil {
		cmdData = make(map[string]interface{})
		json.Unmarshal(message, &cmdData)
	}
	cmdData["message_id"] = messageID
	cmdData["timestamp"] = time.Now().Format(time.RFC3339)
	
	// Agregar client_id y request_id si están disponibles
	if clientID != "" {
		cmdData["client_id"] = clientID
	}
	if requestID != "" {
		cmdData["request_id"] = requestID
	}
	
	// Re-serializar el mensaje con message_id
	messageWithID, err := json.Marshal(cmdData)
	if err != nil {
		return 0, fmt.Errorf("error serializando mensaje: %w", err)
	}

	// Si el caller va a esperar ACK (tiene requestID), creamos canal para notificación asíncrona
	var doneCh chan struct{}
	if requestID != "" {
		doneCh = make(chan struct{})
	}

	// Crear comando pendiente
	pendingCmd := &PendingCommand{
		MessageID:         messageID,
		Command:           messageWithID,
		ScooterID:         scooterID,
		SentAt:            time.Now(),
		RetryCount:        0,
		OriginalTimestamp: cmdData["timestamp"].(string),
		CommandType:       commandType,
		ClientID:          clientID,
		RequestID:         requestID,
		Done:              doneCh,
	}

	// Agregar al mapa de comandos pendientes INMEDIATAMENTE (antes de enviar a cola)
	queue.mu.Lock()
	queue.pendingCommands[messageID] = pendingCmd
	queue.mu.Unlock()

	// Agregar a la cola
	select {
	case queue.queue <- pendingCmd:
		log.Printf("📤 Comando agregado a cola para scooter %d (message_id: %d, tipo: %s)", scooterID, messageID, commandType)
		return messageID, nil
	default:
		queue.mu.Lock()
		delete(queue.pendingCommands, messageID)
		queue.mu.Unlock()
		if doneCh != nil {
			close(doneCh)
		}
		return 0, fmt.Errorf("cola de comandos llena para scooter %d", scooterID)
	}
}

// SendToScooterAndWaitACK envía un mensaje y espera el ACK de forma asíncrona (canal, sin polling).
// Retorna true si se recibió ACK, false si timeout.
func (m *TCPConnectionManager) SendToScooterAndWaitACK(scooterID uint, message []byte, clientID, requestID string, timeout time.Duration) (bool, uint64, error) {
	messageID, err := m.SendToScooter(scooterID, message, clientID, requestID)
	if err != nil {
		return false, 0, err
	}

	m.mu.RLock()
	queue, exists := m.commandQueues[scooterID]
	m.mu.RUnlock()
	if !exists {
		return false, 0, fmt.Errorf("cola de comandos no encontrada")
	}

	queue.mu.RLock()
	cmd, ok := queue.pendingCommands[messageID]
	var doneCh chan struct{}
	if ok && cmd != nil {
		doneCh = cmd.Done
	}
	queue.mu.RUnlock()

	if doneCh == nil {
		// Comando ya terminado (ACK muy rápido) o no encontrado
		queue.mu.RLock()
		_, stillPending := queue.pendingCommands[messageID]
		queue.mu.RUnlock()
		return !stillPending, messageID, nil
	}

	timeoutCh := time.After(timeout)
	select {
	case <-doneCh:
		return true, messageID, nil
	case <-timeoutCh:
		return false, messageID, nil
	}
}

// processCommandQueue procesa la cola de comandos para un scooter
func (m *TCPConnectionManager) processCommandQueue(scooterID uint) {
	m.mu.RLock()
	queue := m.commandQueues[scooterID]
	m.mu.RUnlock()

	if queue == nil {
		return
	}

	for cmd := range queue.queue {
		// Agregar a comandos pendientes
		queue.mu.Lock()
		queue.pendingCommands[cmd.MessageID] = cmd
		queue.mu.Unlock()

		// Enviar comando y esperar ACK
		ackReceived := m.sendCommandAndWaitACK(scooterID, cmd, queue)

		if !ackReceived {
			// Reintentar si no se recibió ACK
			if cmd.RetryCount < m.maxRetries {
				cmd.RetryCount++
				log.Printf("🔄 Reintentando comando para scooter %d (message_id: %d, intento: %d/%d)", 
					scooterID, cmd.MessageID, cmd.RetryCount, m.maxRetries)
				
				// Re-agregar a la cola
				select {
				case queue.queue <- cmd:
				default:
					log.Printf("❌ No se pudo reintentar comando: cola llena")
				}
			} else {
				log.Printf("❌ Comando fallido después de %d intentos (scooter %d, message_id: %d)", 
					m.maxRetries, scooterID, cmd.MessageID)
				// Remover de pendientes y notificar a quien espera
				queue.mu.Lock()
				delete(queue.pendingCommands, cmd.MessageID)
				queue.mu.Unlock()
				if cmd.Done != nil {
					close(cmd.Done)
				}
			}
		} else {
			// ACK recibido, remover de pendientes
			queue.mu.Lock()
			delete(queue.pendingCommands, cmd.MessageID)
			queue.mu.Unlock()
			log.Printf("✅ ACK recibido para comando (scooter %d, message_id: %d)", scooterID, cmd.MessageID)
		}
	}
}

// sendCommandAndWaitACK envía un comando y espera el ACK
func (m *TCPConnectionManager) sendCommandAndWaitACK(scooterID uint, cmd *PendingCommand, queue *CommandQueue) bool {
	m.mu.RLock()
	conn, exists := m.activeConnections[scooterID]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	conn.mu.Lock()
	// Agregar salto de línea al final del mensaje
	messageWithNewline := append(cmd.Command, '\n')
	
	// Configurar timeout de escritura
	conn.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	
	_, err := conn.Conn.Write(messageWithNewline)
	conn.mu.Unlock()

	if err != nil {
		log.Printf("❌ Error enviando comando a scooter %d: %v", scooterID, err)
		return false
	}

	log.Printf("📤 Comando enviado a scooter %d (message_id: %d): %s", scooterID, cmd.MessageID, string(cmd.Command))

	// Esperar ACK con timeout
	timeout := time.NewTimer(m.ackTimeout)
	defer timeout.Stop()

	// Verificar periódicamente si llegó el ACK
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout.C:
			log.Printf("⏱️  Timeout esperando ACK para comando (scooter %d, message_id: %d)", scooterID, cmd.MessageID)
			return false
		case <-ticker.C:
			// Verificar si el comando ya fue marcado como ACK (se elimina del mapa)
			queue.mu.RLock()
			_, stillPending := queue.pendingCommands[cmd.MessageID]
			queue.mu.RUnlock()
			
			if !stillPending {
				// ACK recibido (el comando fue removido del mapa)
				return true
			}
		}
	}
}

// IsScooterOnline verifica si un scooter está conectado
func (m *TCPConnectionManager) IsScooterOnline(scooterID uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.activeConnections[scooterID]
	return exists
}

// GetOnlineScooters retorna la lista de scooters online
func (m *TCPConnectionManager) GetOnlineScooters() []uint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scooters := make([]uint, 0, len(m.activeConnections))
	for scooterID := range m.activeConnections {
		scooters = append(scooters, scooterID)
	}
	return scooters
}

// cleanupInactiveConnections limpia conexiones inactivas periódicamente
func (m *TCPConnectionManager) cleanupInactiveConnections() {
	ticker := time.NewTicker(30 * time.Second) // Verificar cada 30 segundos
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		now := time.Now()
		inactiveScooters := make([]uint, 0)

		for scooterID, conn := range m.activeConnections {
			conn.mu.Lock()
			lastUpdate := conn.LastUpdate
			conn.mu.Unlock()

			if now.Sub(lastUpdate) > m.inactivityTimeout {
				inactiveScooters = append(inactiveScooters, scooterID)
			}
		}
		m.mu.RUnlock()

		// Desregistrar conexiones inactivas
		for _, scooterID := range inactiveScooters {
			log.Printf("⏱️  Scooter %d marcado como offline por inactividad", scooterID)
			m.UnregisterConnection(scooterID)
		}
	}
}

// notifyScooterStatus notifica al frontend sobre el estado de un scooter
func (m *TCPConnectionManager) notifyScooterStatus(scooterID uint, status string) {
	if hub := GetWebSocketHub(); hub != nil {
		message := map[string]interface{}{
			"type":       "scooter_status",
			"scooter_id": scooterID,
			"status":     status,
			"timestamp":  time.Now().Format(time.RFC3339),
		}

		if err := hub.BroadcastMessage(message); err != nil {
			log.Printf("⚠️  Error notificando estado del scooter %d: %v", scooterID, err)
		}
	}
}

// HandleFrontendMessage procesa mensajes del frontend para enviar a scooters
func HandleFrontendMessage(messageData []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(messageData, &msg); err != nil {
		return fmt.Errorf("error parseando mensaje del frontend: %w", err)
	}

	// Extraer client_id del mensaje (agregado automáticamente por ReadPump)
	clientID, _ := msg["_client_id"].(string)
	// Remover _client_id del mensaje antes de procesarlo
	delete(msg, "_client_id")

	action, ok := msg["action"].(string)
	if !ok || action != "send_to_scooter" {
		// No es un mensaje para enviar a un scooter, ignorar
		return nil
	}

	// Obtener scooter_id
	scooterIDFloat, ok := msg["scooter_id"].(float64)
	if !ok {
		return fmt.Errorf("scooter_id inválido o faltante")
	}
	scooterID := uint(scooterIDFloat)

	// Obtener el mensaje a enviar
	messageToSend, ok := msg["message"].(string)
	if !ok {
		// Si no es string, intentar convertir a JSON
		messageBytes, err := json.Marshal(msg["message"])
		if err != nil {
			return fmt.Errorf("error serializando mensaje: %w", err)
		}
		messageToSend = string(messageBytes)
	}

	// Generar request_id único
	requestID := fmt.Sprintf("req_%d_%d", scooterID, time.Now().UnixNano())

	// Enviar mensaje al scooter con client_id y request_id
	manager := GetTCPManager()
	if manager == nil {
		return fmt.Errorf("TCP manager no inicializado")
	}

	if _, err := manager.SendToScooter(scooterID, []byte(messageToSend), clientID, requestID); err != nil {
		// Responder error al frontend (solo al cliente específico si hay client_id)
		if clientID != "" {
			if hub := GetWebSocketHub(); hub != nil {
				errorMsg := map[string]interface{}{
					"type":       "error",
					"scooter_id": scooterID,
					"request_id": requestID,
					"error":      err.Error(),
					"timestamp":  time.Now().Format(time.RFC3339),
				}
				hub.SendToClient(clientID, errorMsg)
			}
		} else {
			// Si no hay client_id, hacer broadcast (fallback)
			if hub := GetWebSocketHub(); hub != nil {
				errorMsg := map[string]interface{}{
					"type":       "error",
					"scooter_id": scooterID,
					"error":      err.Error(),
					"timestamp":  time.Now().Format(time.RFC3339),
				}
				hub.BroadcastMessage(errorMsg)
			}
		}
		return err
	}

	// Responder éxito al frontend (solo al cliente específico si hay client_id)
	if clientID != "" {
		if hub := GetWebSocketHub(); hub != nil {
			successMsg := map[string]interface{}{
				"type":       "message_sent",
				"scooter_id": scooterID,
				"request_id": requestID,
				"status":     "success",
				"timestamp":  time.Now().Format(time.RFC3339),
			}
			hub.SendToClient(clientID, successMsg)
		}
	} else {
		// Si no hay client_id, hacer broadcast (fallback)
		if hub := GetWebSocketHub(); hub != nil {
			successMsg := map[string]interface{}{
				"type":       "message_sent",
				"scooter_id": scooterID,
				"status":     "success",
				"timestamp":  time.Now().Format(time.RFC3339),
			}
			hub.BroadcastMessage(successMsg)
		}
	}

	return nil
}
