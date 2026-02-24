package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// TCPLocationRequest representa la petición TCP para actualizar ubicación
type TCPLocationRequest struct {
	ScooterID   uint     `json:"scooter_id"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	BatteryLevel int     `json:"battery_level"`
	SpeedKmh    *float64 `json:"speed_kmh,omitempty"`
}

// StartTCPServer inicia el servidor TCP que recibe actualizaciones de ubicación
func StartTCPServer(host, port string) error {
	// Asegurar que el TCP manager esté inicializado
	if tcpManager == nil {
		InitTCPManager()
	}

	address := net.JoinHostPort(host, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("error iniciando servidor TCP: %w", err)
	}

	log.Printf("✅ Servidor TCP escuchando en %s", address)
	log.Printf("💡 Formato esperado: {\"scooter_id\": X, \"latitude\": ..., \"longitude\": ..., \"battery_level\": ..., \"speed_kmh\": ...}\\n")
	log.Printf("💡 Conexiones persistentes habilitadas para comunicación bidireccional")

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("❌ Error aceptando conexión TCP: %v", err)
				continue
			}

			// Manejar cada conexión en una goroutine separada
			go handleTCPConnection(conn)
		}
	}()

	return nil
}

// handleTCPConnection maneja una conexión TCP individual
// Mantiene la conexión abierta para permitir comunicación bidireccional
func handleTCPConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("📡 Nueva conexión TCP desde %s", remoteAddr)

	var currentScooterID uint
	manager := GetTCPManager()

	// Configurar timeout de lectura más largo para conexiones persistentes
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// Resetear timeout después de cada lectura
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parsear JSON genérico primero para detectar ACKs
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			log.Printf("❌ Error parseando JSON desde %s: %v", remoteAddr, err)
			log.Printf("   Datos recibidos: %s", line)
			continue
		}

		// Verificar si es un ACK inmediato (múltiples formatos soportados)
		isACK := false
		
		// Formato 1: {"ack": true, ...}
		if ack, ok := jsonData["ack"].(bool); ok && ack {
			isACK = true
		}
		
		// Formato 2: {"type": "ack", ...} (formato del IoT)
		if msgType, ok := jsonData["type"].(string); ok && msgType == "ack" {
			isACK = true
		}
		
		if isACK {
			// Es un ACK inmediato - procesar y continuar
			if manager != nil && currentScooterID != 0 {
				manager.ProcessACK(currentScooterID, jsonData)
			}
			continue // ACK procesado, no es una actualización
		}

		// Si no es ACK, procesar como actualización de ubicación
		var tcpReq TCPLocationRequest
		if err := json.Unmarshal([]byte(line), &tcpReq); err != nil {
			log.Printf("❌ Error parseando actualización desde %s: %v", remoteAddr, err)
			continue
		}

		// Validar que scooter_id esté presente
		if tcpReq.ScooterID == 0 {
			log.Printf("❌ Error: scooter_id es requerido (recibido desde %s)", remoteAddr)
			continue
		}

		// Registrar o actualizar conexión en el manager
		if currentScooterID == 0 || currentScooterID != tcpReq.ScooterID {
			// Primera vez que vemos este scooter_id en esta conexión
			if manager != nil {
				manager.RegisterConnection(tcpReq.ScooterID, conn)
			}
			currentScooterID = tcpReq.ScooterID
		} else {
			// Actualizar timestamp de actividad
			if manager != nil {
				manager.UpdateLastActivity(tcpReq.ScooterID)
			}
		}

		// Convertir a UpdateScooterLocationRequest
		req := UpdateScooterLocationRequest{
			Latitude:    tcpReq.Latitude,
			Longitude:   tcpReq.Longitude,
			BatteryLevel: tcpReq.BatteryLevel,
			SpeedKmh:    tcpReq.SpeedKmh,
		}

		// Procesar actualización usando lógica compartida
		if err := ProcessLocationUpdate(tcpReq.ScooterID, req); err != nil {
			log.Printf("❌ Error procesando actualización desde %s (scooter_id: %d): %v", remoteAddr, tcpReq.ScooterID, err)
			continue
		}

		log.Printf("✅ Actualización procesada desde %s - Scooter ID: %d", remoteAddr, tcpReq.ScooterID)
	}

	// Limpiar cuando la conexión se cierra
	if currentScooterID != 0 && manager != nil {
		manager.UnregisterConnection(currentScooterID)
	}

	if err := scanner.Err(); err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("⏱️  Timeout en conexión TCP desde %s", remoteAddr)
		} else {
			log.Printf("❌ Error leyendo desde %s: %v", remoteAddr, err)
		}
	}

	// Cerrar conexión
	conn.Close()
	log.Printf("🔌 Conexión TCP cerrada desde %s", remoteAddr)
}

// GetTCPPort retorna el puerto TCP desde variables de entorno
func GetTCPPort() string {
	// Buscar TCP_PORT en variables de entorno
	// Si no existe, usar 8201 por defecto
	port := os.Getenv("TCP_PORT")
	if port == "" {
		return "8201"
	}
	return strings.TrimSpace(port)
}

// GetTCPHost retorna el host TCP desde variables de entorno
func GetTCPHost() string {
	// Buscar TCP_HOST en variables de entorno
	// Si no existe, usar "" (todas las interfaces) por defecto
	host := os.Getenv("TCP_HOST")
	return strings.TrimSpace(host)
}
