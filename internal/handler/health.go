package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"ewood-backend/internal/client/postgres"
)

// HealthHandler verifica el estado de las conexiones a las bases de datos
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  make(map[string]string),
	}

	services := health["services"].(map[string]string)

	// Verificar PostgreSQL
	if os.Getenv("POSTGRES_HOST") != "" {
		if err := postgres.Ping(); err == nil {
			services["postgresql"] = "connected"
		} else {
			services["postgresql"] = fmt.Sprintf("disconnected: %v", err)
			health["status"] = "degraded"
		}
	} else {
		services["postgresql"] = "not_configured"
	}

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

