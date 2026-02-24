package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// UpdateScooterLocationRequest representa la petición para actualizar ubicación
type UpdateScooterLocationRequest struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	BatteryLevel int    `json:"battery_level"`
	SpeedKmh    *float64 `json:"speed_kmh,omitempty"`
}

// ProcessLocationUpdate procesa una actualización de ubicación (lógica compartida)
func ProcessLocationUpdate(scooterID uint, req UpdateScooterLocationRequest) error {
	// Validar coordenadas
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("invalid coordinates")
	}

	// Validar batería
	if req.BatteryLevel < 0 || req.BatteryLevel > 100 {
		return fmt.Errorf("invalid battery level (0-100)")
	}

	// Obtener scooter de la base de datos
	var scooter models.Scooter
	if err := postgres.GDB.First(&scooter, scooterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("scooter not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Actualizar ubicación
	scooter.UpdateLocation(req.Latitude, req.Longitude, req.BatteryLevel)
	if err := postgres.GDB.Save(&scooter).Error; err != nil {
		return fmt.Errorf("error updating scooter: %w", err)
	}

	// Guardar ubicación histórica (punto clave)
	location := models.ScooterLocation{
		ScooterID:   scooter.ID,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		EventType:   models.LocationEventPeriodicUpdate,
		BatteryLevel: &req.BatteryLevel,
		SpeedKmh:    req.SpeedKmh,
		RecordedAt:  time.Now(),
	}

	if scooter.CurrentTripID != nil {
		location.TripID = scooter.CurrentTripID
	}

	if err := postgres.GDB.Create(&location).Error; err != nil {
		// No fallar si no se puede guardar la ubicación histórica
		fmt.Printf("⚠️  Error guardando ubicación histórica: %v\n", err)
	}

	// Publicar actualización por WebSocket
	wsMessage := map[string]interface{}{
		"type": "location_update",
		"scooter_id": scooter.ID,
		"data": map[string]interface{}{
			"id":            scooter.ID,
			"code":          scooter.Code,
			"latitude":      req.Latitude,
			"longitude":     req.Longitude,
			"battery_level": req.BatteryLevel,
			"status":        scooter.Status,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}

	// Publicar por WebSocket si el hub está inicializado
	if hub := GetWebSocketHub(); hub != nil {
		if err := hub.BroadcastMessage(wsMessage); err != nil {
			fmt.Printf("⚠️  Error broadcasting WebSocket message: %v\n", err)
		}
	}

	return nil
}

// UpdateScooterLocationHandler maneja la actualización de ubicación de un scooter
func UpdateScooterLocationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	scooterIDStr := vars["id"]
	scooterID, err := strconv.ParseUint(scooterIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid scooter ID", http.StatusBadRequest)
		return
	}

	// Parsear body
	var req UpdateScooterLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Procesar actualización usando lógica compartida
	if err := ProcessLocationUpdate(uint(scooterID), req); err != nil {
		switch err.Error() {
		case "invalid coordinates":
			http.Error(w, "Invalid coordinates", http.StatusBadRequest)
		case "invalid battery level (0-100)":
			http.Error(w, "Invalid battery level (0-100)", http.StatusBadRequest)
		case "scooter not found":
			http.Error(w, "Scooter not found", http.StatusNotFound)
		default:
			http.Error(w, "Error updating location", http.StatusInternalServerError)
		}
		return
	}

	// Responder éxito
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Location updated",
		"scooter_id": scooterID,
	})
}

// UnlockScooterHandler maneja el desbloqueo de un scooter
func UnlockScooterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	scooterIDStr := vars["id"]
	scooterID, err := strconv.ParseUint(scooterIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid scooter ID", http.StatusBadRequest)
		return
	}

	// Verificar que el scooter existe en la base de datos
	var scooter models.Scooter
	if err := postgres.GDB.First(&scooter, uint(scooterID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Scooter not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Obtener el TCP manager
	manager := GetTCPManager()
	if manager == nil {
		http.Error(w, "TCP manager not initialized", http.StatusInternalServerError)
		return
	}

	// Verificar si el scooter está conectado
	if !manager.IsScooterOnline(uint(scooterID)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Scooter is not connected",
			"scooter_id": scooterID,
		})
		return
	}

	// Crear mensaje de unlock en formato JSON
	unlockMessage := map[string]interface{}{
		"command": "unlock",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	messageJSON, err := json.Marshal(unlockMessage)
	if err != nil {
		http.Error(w, "Error creating unlock message", http.StatusInternalServerError)
		return
	}

	// Enviar comando unlock al IoT y esperar ACK
	// Para requests HTTP, no hay client_id (solo para WebSocket)
	// Generar request_id único para tracking
	requestID := fmt.Sprintf("req_%d_%d", scooterID, time.Now().UnixNano())
	
	// Esperar ACK con timeout de 25 segundos (menor que el timeout del manager de 30s para dar margen)
	ackReceived, messageID, err := manager.SendToScooterAndWaitACK(uint(scooterID), messageJSON, "", requestID, 25*time.Second)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"scooter_id": scooterID,
		})
		return
	}

	// Responder según si se recibió ACK o no
	w.Header().Set("Content-Type", "application/json")
	if ackReceived {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Unlock command sent and acknowledged",
			"scooter_id": scooterID,
			"message_id": messageID,
			"request_id": requestID,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	} else {
		// Comando enviado pero ACK no recibido en el timeout
		w.WriteHeader(http.StatusAccepted) // 202 Accepted - comando aceptado pero no confirmado
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Unlock command sent, waiting for acknowledgment",
			"scooter_id": scooterID,
			"message_id": messageID,
			"request_id": requestID,
			"ack_received": false,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// LockScooterHandler maneja el bloqueo de un scooter
func LockScooterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	scooterIDStr := vars["id"]
	scooterID, err := strconv.ParseUint(scooterIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid scooter ID", http.StatusBadRequest)
		return
	}

	// Verificar que el scooter existe en la base de datos
	var scooter models.Scooter
	if err := postgres.GDB.First(&scooter, uint(scooterID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Scooter not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Obtener el TCP manager
	manager := GetTCPManager()
	if manager == nil {
		http.Error(w, "TCP manager not initialized", http.StatusInternalServerError)
		return
	}

	// Verificar si el scooter está conectado
	if !manager.IsScooterOnline(uint(scooterID)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"error":     "Scooter is not connected",
			"scooter_id": scooterID,
		})
		return
	}

	// Crear mensaje de lock en formato JSON
	lockMessage := map[string]interface{}{
		"command":   "lock",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	messageJSON, err := json.Marshal(lockMessage)
	if err != nil {
		http.Error(w, "Error creating lock message", http.StatusInternalServerError)
		return
	}

	// Enviar comando lock al IoT y esperar ACK
	requestID := fmt.Sprintf("req_%d_%d", scooterID, time.Now().UnixNano())
	ackReceived, messageID, err := manager.SendToScooterAndWaitACK(uint(scooterID), messageJSON, "", requestID, 25*time.Second)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"scooter_id": scooterID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if ackReceived {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"message":    "Lock command sent and acknowledged",
			"scooter_id": scooterID,
			"message_id": messageID,
			"request_id": requestID,
			"timestamp":  time.Now().Format(time.RFC3339),
		})
	} else {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"message":    "Lock command sent, waiting for acknowledgment",
			"scooter_id":  scooterID,
			"message_id":  messageID,
			"request_id":  requestID,
			"ack_received": false,
			"timestamp":   time.Now().Format(time.RFC3339),
		})
	}
}

