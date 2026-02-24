package router

import (
	"net/http"

	"ewood-backend/internal/handler"

	"github.com/gorilla/mux"
)

// NewRouter crea y configura el router HTTP
func NewRouter() http.Handler {
	r := mux.NewRouter()

	// Rutas básicas
	r.HandleFunc("/check", handler.CheckHandler).Methods("GET")
	r.HandleFunc("/health", handler.HealthHandler).Methods("GET")
	r.HandleFunc("/env", handler.EnvHandler).Methods("GET")

	// WebSocket
	r.HandleFunc("/ws/scooters", handler.WebSocketHandler)

	// API de Scooters
	api := r.PathPrefix("/api").Subrouter()
	
	// Admin endpoints (requieren X-Admin-Key header)
	api.HandleFunc("/admin/scooters", handler.CreateScooterHandler).Methods("POST")
	
	// Scooters endpoints públicos
	api.HandleFunc("/scooters", handler.ListScootersHandler).Methods("GET")
	api.HandleFunc("/scooters/{id}", handler.GetScooterHandler).Methods("GET")
	api.HandleFunc("/scooters/{id}/location", handler.UpdateScooterLocationHandler).Methods("POST")
	api.HandleFunc("/scooters/{id}/unlock", handler.UnlockScooterHandler).Methods("POST")
	api.HandleFunc("/scooters/{id}/lock", handler.LockScooterHandler).Methods("POST")

	// Aplicar CORS a todas las rutas
	return CORS(DefaultCORSConfig())(r)
}

