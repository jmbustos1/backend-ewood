package router

import (
	"net/http"

	"ewood-backend/internal/handler"
	"ewood-backend/internal/middleware"

	"github.com/gorilla/mux"
)

// NewRouter crea y configura el router HTTP
func NewRouter() http.Handler {
	r := mux.NewRouter()

	// ── Rutas públicas ──────────────────────────────────────────────────────────
	r.HandleFunc("/check", handler.CheckHandler).Methods("GET")
	r.HandleFunc("/health", handler.HealthHandler).Methods("GET")
	r.HandleFunc("/env", handler.EnvHandler).Methods("GET")

	// Auth (públicos)
	r.HandleFunc("/auth/register", handler.RegisterHandler).Methods("POST")
	r.HandleFunc("/auth/login", handler.LoginHandler).Methods("POST")
	r.HandleFunc("/auth/refresh", handler.RefreshTokenHandler).Methods("POST")
	r.HandleFunc("/auth/forgot-password", handler.ForgotPasswordHandler).Methods("POST")

	// Ciudades (público — necesario en onboarding antes de login)
	r.HandleFunc("/api/cities", handler.ListCitiesHandler).Methods("GET")

	// IoT — actualización de ubicación (protegida por device secret en producción)
	r.HandleFunc("/api/scooters/{id}/location", handler.UpdateScooterLocationHandler).Methods("POST")

	// Admin
	r.HandleFunc("/api/admin/scooters", handler.CreateScooterHandler).Methods("POST")

	// Webhook MercadoPago (público — MP llama desde sus servidores sin JWT).
	// Seguridad: el handler verifica el pago contra la API de MP con nuestro
	// Access Token antes de acreditar saldo. Idempotente por mp_payment_id.
	r.HandleFunc("/api/wallet/mp-webhook", handler.MPWebhookHandler).Methods("POST")

	// ── Rutas protegidas (JWT requerido) ────────────────────────────────────────
	protected := r.NewRoute().Subrouter()
	protected.Use(middleware.RequireAuth)

	// Auth protegido
	protected.HandleFunc("/auth/me", handler.MeHandler).Methods("GET")
	protected.HandleFunc("/auth/logout", handler.LogoutHandler).Methods("POST")

	// WebSocket (token en query param: /ws/scooters?token=xxx)
	protected.HandleFunc("/ws/scooters", handler.WebSocketHandler)

	// Scooters
	protected.HandleFunc("/api/scooters", handler.ListScootersHandler).Methods("GET")
	protected.HandleFunc("/api/scooters/{id}", handler.GetScooterHandler).Methods("GET")
	protected.HandleFunc("/api/scooters/{id}/unlock", handler.UnlockScooterHandler).Methods("POST")
	protected.HandleFunc("/api/scooters/{id}/lock", handler.LockScooterHandler).Methods("POST")

	// Viajes
	protected.HandleFunc("/api/trips", handler.StartTripHandler).Methods("POST")
	protected.HandleFunc("/api/trips", handler.ListTripsHandler).Methods("GET")
	protected.HandleFunc("/api/trips/active", handler.GetActiveTripHandler).Methods("GET")
	protected.HandleFunc("/api/trips/{id}", handler.GetTripHandler).Methods("GET")
	protected.HandleFunc("/api/trips/{id}/end", handler.EndTripHandler).Methods("PATCH")

	// Billetera (saldo prepago)
	protected.HandleFunc("/api/wallet", handler.GetWalletHandler).Methods("GET")
	protected.HandleFunc("/api/wallet/recharge", handler.RechargeWalletHandler).Methods("POST")

	// Aplicar CORS a todas las rutas
	return CORS(DefaultCORSConfig())(r)
}
