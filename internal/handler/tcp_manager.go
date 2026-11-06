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
	Done              chan struct{} // se cierra UNA vez cuando el comando termina (ACK, fallo, expirado o abandonado)

	finished atomic.Bool // true cuando el comando terminó; no se reenvía ni reintenta
	acked    atomic.Bool // true si terminó por ACK real del scooter
}

// CommandQueue gestiona la cola de comandos para un scooter.
//
// queue nunca se cierra con close(): escribir en un canal cerrado hace panic
// y tumba el proceso entero (incidente 2026-11-06 02:44 UTC). Para detener la
// cola se cierra closed, que el worker y los productores consultan.
type CommandQueue struct {
	queue            chan *PendingCommand
	pendingCommands  map[uint64]*PendingCommand // message_id -> comando
	mu               sync.RWMutex
	messageIDCounter uint64
	closed           chan struct{}
	closeOnce        sync.Once
}

func newCommandQueue() *CommandQueue {
	return &CommandQueue{
		queue:           make(chan *PendingCommand, 100), // Buffer de 100 comandos
		pendingCommands: make(map[uint64]*PendingCommand),
		closed:          make(chan struct{}),
	}
}

// stop señala que la cola no recibe más comandos. Idempotente.
func (q *CommandQueue) stop() {
	q.closeOnce.Do(func() { close(q.closed) })
}

// finish termina un comando exactamente una vez: lo saca de pendientes, marca
// si fue ACK y cierra Done. Retorna false si ya estaba terminado.
func (q *CommandQueue) finish(cmd *PendingCommand, acked bool) bool {
	if !cmd.finished.CompareAndSwap(false, true) {
		return false
	}
	if acked {
		cmd.acked.Store(true)
	}
	q.mu.Lock()
	delete(q.pendingCommands, cmd.MessageID)
	q.mu.Unlock()
	if cmd.Done != nil {
		close(cmd.Done)
	}
	return true
}

// failAll termina como fallidos todos los comandos pendientes (al desconectar).
func (q *CommandQueue) failAll() {
	q.mu.RLock()
	cmds := make([]*PendingCommand, 0, len(q.pendingCommands))
	for _, cmd := range q.pendingCommands {
		cmds = append(cmds, cmd)
	}
	q.mu.RUnlock()
	for _, cmd := range cmds {
		q.finish(cmd, false)
	}
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
		queue := newCommandQueue()
		m.commandQueues[scooterID] = queue
		// Iniciar goroutine para procesar la cola
		go m.processCommandQueue(scooterID, queue)
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

// UnregisterConnection elimina la conexión del mapa solo si sigue siendo conn.
// Si el scooter ya se reconectó con un socket nuevo, la goroutine del socket
// viejo no debe tumbar la conexión nueva (T-0060).
func (m *TCPConnectionManager) UnregisterConnection(scooterID uint, conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, exists := m.activeConnections[scooterID]; exists && c.Conn == conn {
		delete(m.activeConnections, scooterID)
		log.Printf("❌ Scooter %d desconectado. Total activos: %d", scooterID, len(m.activeConnections))

		// Detener la cola: el worker termina y falla los comandos pendientes
		if queue, queueExists := m.commandQueues[scooterID]; queueExists {
			queue.stop()
			delete(m.commandQueues, scooterID)
		}
		
		// Notificar al frontend que el scooter está offline
		m.notifyScooterStatus(scooterID, "offline")
		
		// Cerrar conexión
		c.mu.Lock()
		c.Conn.Close()
		c.mu.Unlock()
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

	// Buscar y terminar el comando pendiente (notifica a SendToScooterAndWaitACK)
	queue.mu.RLock()
	cmd, cmdExists := queue.pendingCommands[messageID]
	queue.mu.RUnlock()

	if !cmdExists || !queue.finish(cmd, true) {
		// Comando desconocido o ya terminado (p. ej. abandonado por timeout).
		// Si era un unlock abandonado, el scooter pudo quedar desbloqueado sin viaje.
		log.Printf("⚠️  ACK recibido para comando desconocido o ya terminado (scooter %d, message_id: %d, request_id: %s)",
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
			queue.mu.RLock()
			var expiredCmds []*PendingCommand
			for _, cmd := range queue.pendingCommands {
				if now.Sub(cmd.SentAt) > (m.ackTimeout * 2) {
					expiredCmds = append(expiredCmds, cmd)
				}
			}
			queue.mu.RUnlock()
			for _, cmd := range expiredCmds {
				if queue.finish(cmd, false) {
					log.Printf("⏱️  Comando expirado removido (scooter %d, message_id: %d)", scooterID, cmd.MessageID)
				}
			}
		}
	}
}

// SendToScooter envía un mensaje a un scooter específico usando la cola.
// Retorna el message_id asignado (para correlación/espera de ACK).
// clientID y requestID son opcionales para correlación con clientes WebSocket.
func (m *TCPConnectionManager) SendToScooter(scooterID uint, message []byte, clientID, requestID string) (uint64, error) {
	cmd, _, err := m.enqueue(scooterID, message, clientID, requestID)
	if err != nil {
		return 0, err
	}
	return cmd.MessageID, nil
}

// enqueue arma el comando con message_id y lo agrega a la cola del scooter.
func (m *TCPConnectionManager) enqueue(scooterID uint, message []byte, clientID, requestID string) (*PendingCommand, *CommandQueue, error) {
	m.mu.RLock()
	_, exists := m.activeConnections[scooterID]
	queue, queueExists := m.commandQueues[scooterID]
	m.mu.RUnlock()

	if !exists {
		return nil, nil, fmt.Errorf("scooter %d no está conectado", scooterID)
	}

	if !queueExists {
		return nil, nil, fmt.Errorf("cola de comandos no inicializada para scooter %d", scooterID)
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
		return nil, nil, fmt.Errorf("error serializando mensaje: %w", err)
	}

	// Crear comando pendiente. Done siempre existe: finish() lo cierra una vez.
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
		Done:              make(chan struct{}),
	}

	// Agregar al mapa de comandos pendientes INMEDIATAMENTE (antes de enviar a cola)
	queue.mu.Lock()
	queue.pendingCommands[messageID] = pendingCmd
	queue.mu.Unlock()

	// Agregar a la cola (nunca está cerrada con close(), ver CommandQueue)
	select {
	case <-queue.closed:
		queue.finish(pendingCmd, false)
		return nil, nil, fmt.Errorf("scooter %d no está conectado", scooterID)
	case queue.queue <- pendingCmd:
		log.Printf("📤 Comando agregado a cola para scooter %d (message_id: %d, tipo: %s)", scooterID, messageID, commandType)
		return pendingCmd, queue, nil
	default:
		queue.finish(pendingCmd, false)
		return nil, nil, fmt.Errorf("cola de comandos llena para scooter %d", scooterID)
	}
}

// SendToScooterAndWaitACK envía un mensaje y espera el ACK de forma asíncrona (canal, sin polling).
// Retorna true solo si llegó un ACK real. Si se agota timeout, el comando
// queda abandonado: el worker no lo reintenta (evita, p. ej., un unlock
// reenviado después de que el viaje ya se canceló).
func (m *TCPConnectionManager) SendToScooterAndWaitACK(scooterID uint, message []byte, clientID, requestID string, timeout time.Duration) (bool, uint64, error) {
	cmd, queue, err := m.enqueue(scooterID, message, clientID, requestID)
	if err != nil {
		return false, 0, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-cmd.Done:
		return cmd.acked.Load(), cmd.MessageID, nil
	case <-timer.C:
		if queue.finish(cmd, false) {
			log.Printf("🛑 Comando abandonado tras %s sin ACK (scooter %d, message_id: %d, tipo: %s)",
				timeout, scooterID, cmd.MessageID, cmd.CommandType)
			return false, cmd.MessageID, nil
		}
		// Terminó justo al mismo tiempo (ACK o fallo)
		return cmd.acked.Load(), cmd.MessageID, nil
	}
}

// processCommandQueue procesa la cola de comandos de un scooter hasta que la
// cola se detiene (desconexión).
func (m *TCPConnectionManager) processCommandQueue(scooterID uint, queue *CommandQueue) {
	for {
		var cmd *PendingCommand
		select {
		case <-queue.closed:
			queue.failAll()
			return
		case cmd = <-queue.queue:
		}

		// Abandonado, expirado o ya con ACK mientras esperaba en la cola
		if cmd.finished.Load() {
			continue
		}

		// Enviar comando y esperar ACK
		if m.sendCommandAndWaitACK(scooterID, cmd, queue) {
			log.Printf("✅ ACK recibido para comando (scooter %d, message_id: %d)", scooterID, cmd.MessageID)
			continue
		}

		if cmd.finished.Load() {
			log.Printf("🛑 Comando descartado, sin reintento (scooter %d, message_id: %d): abandonado, expirado o scooter desconectado",
				scooterID, cmd.MessageID)
			continue
		}

		if cmd.RetryCount >= m.maxRetries {
			log.Printf("❌ Comando fallido después de %d intentos (scooter %d, message_id: %d)",
				m.maxRetries, scooterID, cmd.MessageID)
			queue.finish(cmd, false)
			continue
		}

		cmd.RetryCount++
		log.Printf("🔄 Reintentando comando para scooter %d (message_id: %d, intento: %d/%d)",
			scooterID, cmd.MessageID, cmd.RetryCount, m.maxRetries)

		// Re-agregar a la cola
		select {
		case <-queue.closed:
			queue.finish(cmd, false)
		case queue.queue <- cmd:
		default:
			log.Printf("❌ No se pudo reintentar comando: cola llena")
			queue.finish(cmd, false)
		}
	}
}

// sendCommandAndWaitACK envía un comando y espera el ACK. Retorna true solo
// con ACK real; false por timeout, error de escritura, desconexión o si el
// comando fue terminado por otro lado (abandonado/expirado).
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

	select {
	case <-cmd.Done:
		return cmd.acked.Load()
	case <-queue.closed:
		return false
	case <-timeout.C:
		log.Printf("⏱️  Timeout esperando ACK para comando (scooter %d, message_id: %d)", scooterID, cmd.MessageID)
		return false
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
		inactive := make(map[uint]net.Conn)

		for scooterID, conn := range m.activeConnections {
			conn.mu.Lock()
			lastUpdate := conn.LastUpdate
			conn.mu.Unlock()

			if now.Sub(lastUpdate) > m.inactivityTimeout {
				inactive[scooterID] = conn.Conn
			}
		}
		m.mu.RUnlock()

		// Desregistrar conexiones inactivas (solo si siguen siendo las mismas)
		for scooterID, conn := range inactive {
			log.Printf("⏱️  Scooter %d marcado como offline por inactividad", scooterID)
			m.UnregisterConnection(scooterID, conn)
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
