package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/models"

	"github.com/gorilla/mux"
)

// CreateScooterRequest representa la petición para crear un scooter
type CreateScooterRequest struct {
	Code         string  `json:"code"`
	Model        string  `json:"model,omitempty"`
	AuthKey      string  `json:"auth_key"` // Clave de autenticación del scooter
}

// CreateScooterHandler crea un nuevo scooter (requiere clave de administración)
func CreateScooterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verificar clave de administración
	adminKey := r.Header.Get("X-Admin-Key")
	expectedKey := os.Getenv("SCOOTER_ADMIN_KEY")
	
	if expectedKey == "" {
		http.Error(w, "Admin key not configured", http.StatusInternalServerError)
		return
	}

	if adminKey != expectedKey {
		http.Error(w, "Unauthorized: Invalid admin key", http.StatusUnauthorized)
		return
	}

	// Parsear body
	var req CreateScooterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validar campos requeridos
	if req.Code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	if req.AuthKey == "" {
		http.Error(w, "Auth key is required", http.StatusBadRequest)
		return
	}

	// Verificar que el código no exista
	var existing models.Scooter
	if err := postgres.GDB.Where("code = ?", req.Code).First(&existing).Error; err == nil {
		http.Error(w, "Scooter with this code already exists", http.StatusConflict)
		return
	}

	// Crear scooter
	scooter := models.Scooter{
		Code:         req.Code,
		Model:        req.Model,
		Status:       models.ScooterStatusAvailable,
		BatteryLevel: 100,
	}

	if err := postgres.GDB.Create(&scooter).Error; err != nil {
		http.Error(w, "Error creating scooter", http.StatusInternalServerError)
		return
	}

	// TODO: Guardar auth_key en una tabla separada o en el scooter
	// Por ahora, la auth_key se valida al recibir actualizaciones de ubicación

	// Responder éxito
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Scooter created",
		"scooter": scooter,
	})
}

// ListScootersHandler lista todos los scooters, con filtro opcional por city_id
func ListScootersHandler(w http.ResponseWriter, r *http.Request) {
	query := postgres.GDB.Where("deleted_at IS NULL")

	if cityID := r.URL.Query().Get("city_id"); cityID != "" {
		query = query.Where("city_id = ?", cityID)
	}

	scooters := make([]models.Scooter, 0)
	if err := query.Find(&scooters).Error; err != nil {
		http.Error(w, "Error fetching scooters", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, scooters)
}

// GetScooterHandler obtiene un scooter por ID
func GetScooterHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scooterIDStr := vars["id"]

	var scooter models.Scooter
	if err := postgres.GDB.First(&scooter, scooterIDStr).Error; err != nil {
		http.Error(w, "Scooter not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"scooter": scooter,
	})
}

