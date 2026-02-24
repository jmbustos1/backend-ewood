package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// CheckHandler maneja las peticiones al endpoint /check
func CheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    "ok",
		"message":   "Servidor funcionando correctamente",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

