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
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	BatteryLevel int      `json:"battery_level"`
	SpeedKmh     *float64 `json:"speed_kmh,omitempty"`
	// LockState del actuador reportado por el firmware. Valores esperados:
	// "locked" | "unlocked" | "unknown". Fuente de verdad del estado fisico.
	LockState string `json:"lock_state,omitempty"`
	// Diag: bloque "d" de diagnostico del firmware (solo por TCP).
	Diag json.RawMessage `json:"d,omitempty"`
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

	// Actualizar ubicación (conserva la última posición real si llega 0,0)
	positionChanged := scooter.UpdateLocation(req.Latitude, req.Longitude, req.BatteryLevel)

	// Solo columnas de telemetría: un Save() de la fila completa pisaría
	// status/current_trip_id si un viaje se inicia o cierra en paralelo.
	updates := map[string]interface{}{
		"battery_level": scooter.BatteryLevel,
	}
	if positionChanged {
		updates["last_latitude"] = scooter.LastLatitude
		updates["last_longitude"] = scooter.LastLongitude
		updates["last_location_updated_at"] = scooter.LastLocationUpdatedAt
	}

	// Actualizar lock_state si viene en el request y es un valor conocido.
	// Los scooters que aun no reportan (firmware viejo o SYS STATUS no recibido)
	// mantienen su valor previo — no lo reseteamos a 'unknown' por default.
	if req.LockState != "" {
		newLockState := models.LockState(req.LockState)
		if newLockState == models.LockStateLocked ||
			newLockState == models.LockStateUnlocked ||
			newLockState == models.LockStateUnknown {
			if scooter.LockState != newLockState {
				scooter.LockState = newLockState
				now := time.Now()
				scooter.LockStateUpdatedAt = &now
				updates["lock_state"] = scooter.LockState
				updates["lock_state_updated_at"] = scooter.LockStateUpdatedAt
			}
		}
	}

	if diag, ok := parseDiag(req.Diag); ok {
		logDiagChanges(scooter.ID, scooter.LastDiag, diag)
		raw := string(req.Diag)
		now := time.Now()
		updates["last_diag"] = raw
		updates["last_diag_at"] = now
	}

	if err := postgres.GDB.Model(&scooter).Updates(updates).Error; err != nil {
		return fmt.Errorf("error updating scooter: %w", err)
	}

	// Guardar ubicación histórica (punto clave). Los 0,0 sin fix no se guardan:
	// ensucian el recorrido del viaje.
	if !models.IsNoFix(req.Latitude, req.Longitude) {
		location := models.ScooterLocation{
			ScooterID:    scooter.ID,
			Latitude:     req.Latitude,
			Longitude:    req.Longitude,
			EventType:    models.LocationEventPeriodicUpdate,
			BatteryLevel: &req.BatteryLevel,
			SpeedKmh:     req.SpeedKmh,
			RecordedAt:   time.Now(),
		}

		if scooter.CurrentTripID != nil {
			location.TripID = scooter.CurrentTripID
		}

		if err := postgres.GDB.Create(&location).Error; err != nil {
			// No fallar si no se puede guardar la ubicación histórica
			fmt.Printf("⚠️  Error guardando ubicación histórica: %v\n", err)
		}
	}

	// Publicar actualización por WebSocket con la posición guardada (no el 0,0
	// crudo), para que el mapa no pierda el scooter mientras no tiene fix.
	// UpdateLocation garantiza que LastLatitude/LastLongitude no son nil aquí.
	wsMessage := map[string]interface{}{
		"type": "location_update",
		"scooter_id": scooter.ID,
		"data": map[string]interface{}{
			"id":            scooter.ID,
			"code":          scooter.Code,
			"latitude":      *scooter.LastLatitude,
			"longitude":     *scooter.LastLongitude,
			"battery_level": req.BatteryLevel,
			"status":        scooter.Status,
			"lock_state":    scooter.LockState,
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
