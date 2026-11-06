package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/middleware"
	"ewood-backend/internal/models"
)

// ─── Request types ────────────────────────────────────────────────────────────

type startTripRequest struct {
	ScooterID uint `json:"scooter_id"`
}

type endTripRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ─── Constants ──────────────────────────────────────────────────────────────

// TODO: volver a 15 cuando el BMS esté conectado
const minBatteryForRental = 0 // TEMPORAL: deshabilitado para pruebas sin BMS

// defaultPricePerMinute se usa si la env var PRICE_PER_MINUTE no está seteada (CLP, neto).
const defaultPricePerMinute = 150.0

// ivaRate es el IVA chileno (19%). Aplicado sobre el subtotal del viaje.
const ivaRate = 0.19

// getPricePerMinute lee el precio neto por minuto desde env o retorna el default.
func getPricePerMinute() float64 {
	raw := os.Getenv("PRICE_PER_MINUTE")
	if raw == "" {
		return defaultPricePerMinute
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return defaultPricePerMinute
	}
	return v
}

// tripCostBreakdown representa el desglose de cobro de un viaje.
type tripCostBreakdown struct {
	MinutesCharged int
	PricePerMinute float64
	Subtotal       float64
	TaxAmount      float64
	Total          float64
}

// calculateTripCost calcula el desglose de costo de un viaje según su duración.
// Cobra por minuto iniciado (ceil) — un viaje de 1m1s cuesta 2 minutos.
// Subtotal = minutos × precio_neto.  IVA = subtotal × 19%.  Total = subtotal + IVA.
// Montos redondeados a pesos enteros (CLP no tiene decimales).
// Mobile replica esta fórmula en src/domain/usecases/calculateTripCost.ts.
func calculateTripCost(durationSeconds int) tripCostBreakdown {
	price := getPricePerMinute()
	if durationSeconds <= 0 {
		return tripCostBreakdown{PricePerMinute: price}
	}
	minutes := int(math.Ceil(float64(durationSeconds) / 60.0))
	subtotal := math.Round(float64(minutes) * price)
	tax := math.Round(subtotal * ivaRate)
	return tripCostBreakdown{
		MinutesCharged: minutes,
		PricePerMinute: price,
		Subtotal:       subtotal,
		TaxAmount:      tax,
		Total:          subtotal + tax,
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func formatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func formatDistance(meters float64) string {
	if meters < 1000 {
		return fmt.Sprintf("%.0f m", meters)
	}
	return fmt.Sprintf("%.1f km", meters/1000)
}

// haversine calcula distancia en metros entre dos coordenadas
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// sendScooterCommand envia un comando lock/unlock al scooter via TCP y espera ACK.
// Retorna (ackReceived, messageID, error).
func sendScooterCommand(scooterID uint, command string) (bool, uint64, error) {
	manager := GetTCPManager()
	if manager == nil {
		return false, 0, fmt.Errorf("tcp_manager_not_initialized")
	}

	if !manager.IsScooterOnline(scooterID) {
		return false, 0, fmt.Errorf("scooter_not_connected")
	}

	msg := map[string]interface{}{
		"command":   command,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return false, 0, fmt.Errorf("json_marshal_error")
	}

	requestID := fmt.Sprintf("req_%d_%d", scooterID, time.Now().UnixNano())
	ack, messageID, err := manager.SendToScooterAndWaitACK(scooterID, msgJSON, "", requestID, 25*time.Second)
	return ack, messageID, err
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

func StartTripHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ScooterID == 0 {
		writeError(w, http.StatusBadRequest, "scooter_id is required")
		return
	}

	// ── Transaccion con FOR UPDATE para evitar race conditions ────────────
	tx := postgres.GDB.Begin()
	if tx.Error != nil {
		writeError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}

	// Lock de fila: si dos requests llegan al mismo tiempo, uno espera
	var scooter models.Scooter
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&scooter, req.ScooterID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			writeError(w, http.StatusNotFound, "scooter_not_found")
		} else {
			writeError(w, http.StatusInternalServerError, "database_error")
		}
		return
	}

	// ── Validaciones de negocio (rechazo inmediato, sin TCP) ─────────────

	if !scooter.IsAvailable() {
		tx.Rollback()
		writeError(w, http.StatusConflict, "scooter_not_available")
		return
	}

	if scooter.BatteryLevel < minBatteryForRental {
		tx.Rollback()
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("battery_too_low: %d%% (min %d%%)", scooter.BatteryLevel, minBatteryForRental))
		return
	}

	manager := GetTCPManager()
	if manager == nil || !manager.IsScooterOnline(req.ScooterID) {
		tx.Rollback()
		writeError(w, http.StatusServiceUnavailable, "scooter_not_connected")
		return
	}

	// Verificar que el usuario no tiene un viaje activo
	var existing models.Trip
	if err := tx.Where("user_id = ? AND status = ?", userID, models.TripStatusActive).First(&existing).Error; err == nil {
		tx.Rollback()
		writeError(w, http.StatusConflict, "you_already_have_an_active_trip")
		return
	}

	// Validar saldo de la wallet (no permitir arriendo con deuda)
	wallet, err := getOrCreateWallet(tx, userID)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "could_not_load_wallet")
		return
	}
	if !wallet.CanRent() {
		tx.Rollback()
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"error":   "negative_balance",
			"balance": wallet.Balance,
			"message": "Saldo insuficiente. Recarga tu billetera para continuar.",
		})
		return
	}

	// ── Crear trip y marcar scooter en uso (dentro de la transaccion) ────

	lat := 0.0
	lng := 0.0
	if scooter.LastLatitude != nil {
		lat = *scooter.LastLatitude
	}
	if scooter.LastLongitude != nil {
		lng = *scooter.LastLongitude
	}

	trip := models.Trip{
		ScooterID:      req.ScooterID,
		UserID:         userID,
		StartLatitude:  lat,
		StartLongitude: lng,
		Status:         models.TripStatusActive,
	}

	if err := tx.Create(&trip).Error; err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "could_not_create_trip")
		return
	}

	now := time.Now()
	if err := tx.Model(&scooter).Updates(map[string]interface{}{
		"status":          models.ScooterStatusInUse,
		"current_user_id": userID,
		"current_trip_id": trip.ID,
		"trip_started_at": now,
	}).Error; err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "could_not_update_scooter")
		return
	}

	// Commit la transaccion ANTES de enviar TCP
	if err := tx.Commit().Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_commit")
		return
	}

	// ── Enviar unlock al hardware ────────────────────────────────────────

	ack, _, err := sendScooterCommand(req.ScooterID, "unlock")

	if err != nil || !ack {
		// Rollback manual: el hardware no respondio
		postgres.GDB.Model(&models.Scooter{}).Where("id = ?", req.ScooterID).Updates(map[string]interface{}{
			"status":          models.ScooterStatusAvailable,
			"current_user_id": nil,
			"current_trip_id": nil,
			"trip_started_at": nil,
		})
		postgres.GDB.Model(&models.Trip{}).Where("id = ?", trip.ID).Update("status", models.TripStatusCancelled)

		errMsg := "scooter_did_not_respond"
		if err != nil {
			errMsg = err.Error()
		}
		writeError(w, http.StatusServiceUnavailable, errMsg)
		return
	}

	// ── Exito ────────────────────────────────────────────────────────────

	writeJSON(w, http.StatusCreated, map[string]any{
		"trip":       trip,
		"scooter_id": req.ScooterID,
		"message":    "trip_started",
	})
}

func EndTripHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tripID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return
	}

	var req endTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var trip models.Trip
	if err := postgres.GDB.First(&trip, tripID).Error; err != nil {
		writeError(w, http.StatusNotFound, "trip not found")
		return
	}
	if trip.UserID != userID {
		writeError(w, http.StatusForbidden, "not your trip")
		return
	}
	if !trip.IsActive() {
		// Idempotente: si el trip ya se completó, retornar el resumen en vez de error.
		// Cubre el caso donde la app no recibió la respuesta original (network error)
		// y reintenta — el backend ya procesó todo, solo falta confirmar al cliente.
		if trip.Status == models.TripStatusCompleted {
			durationSecs := 0
			if trip.DurationSeconds != nil {
				durationSecs = *trip.DurationSeconds
			}
			distanceMeters := 0.0
			if trip.DistanceMeters != nil {
				distanceMeters = *trip.DistanceMeters
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"trip":               trip,
				"duration_formatted": formatDuration(durationSecs),
				"distance_formatted": formatDistance(distanceMeters),
				"lock_sent":          true,
				"already_completed":  true,
			})
			return
		}
		writeError(w, http.StatusConflict, "trip is not active")
		return
	}

	// ── Enviar lock al hardware ──────────────────────────────────────────

	ack, _, cmdErr := sendScooterCommand(trip.ScooterID, "lock")
	lockSent := ack && cmdErr == nil

	// ── Calcular costo, completar trip, cobrar wallet (todo en transaccion) ────

	now := time.Now()
	durationSecs := int(now.Sub(trip.StartedAt).Seconds())
	phoneHasFix := !models.IsNoFix(req.Lat, req.Lng) &&
		req.Lat >= -90 && req.Lat <= 90 && req.Lng >= -180 && req.Lng <= 180

	// Si el viaje partió sin fix (0,0), la distancia contra el teléfono sería
	// absurda (miles de km): se deja en 0.
	distanceMeters := 0.0
	if phoneHasFix && !models.IsNoFix(trip.StartLatitude, trip.StartLongitude) {
		distanceMeters = haversine(trip.StartLatitude, trip.StartLongitude, req.Lat, req.Lng)
	}
	costBreakdown := calculateTripCost(durationSecs)

	tx := postgres.GDB.Begin()
	if tx.Error != nil {
		writeError(w, http.StatusInternalServerError, "could_not_start_transaction")
		return
	}

	// trip.CostAmount guarda el TOTAL (con IVA) — es lo que se descuenta al usuario.
	trip.Complete(req.Lat, req.Lng, distanceMeters, 0, 0, costBreakdown.Total)
	if err := tx.Save(&trip).Error; err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "could_not_save_trip")
		return
	}

	scooterUpdates := map[string]interface{}{
		"status":          models.ScooterStatusAvailable,
		"current_user_id": nil,
		"current_trip_id": nil,
		"trip_started_at": nil,
	}

	// Si el scooter no tiene fix GPS, usar la posición del teléfono: el usuario
	// lo deja justo ahí, así queda visible en el mapa donde lo dejaron.
	var scooter models.Scooter
	if err := tx.Select("id", "last_latitude", "last_longitude").First(&scooter, trip.ScooterID).Error; err == nil &&
		!scooter.HasValidFix() && phoneHasFix {
		scooterUpdates["last_latitude"] = req.Lat
		scooterUpdates["last_longitude"] = req.Lng
		scooterUpdates["last_location_updated_at"] = now
		log.Printf("📍 Scooter %d sin fix GPS: se guarda la posición del teléfono al cerrar el viaje %d", trip.ScooterID, trip.ID)
	}

	if err := tx.Model(&models.Scooter{}).Where("id = ?", trip.ScooterID).Updates(scooterUpdates).Error; err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "could_not_release_scooter")
		return
	}

	// Cobrar al wallet el TOTAL con IVA (amount negativo). Puede dejar balance
	// negativo (deuda).
	tripIDPtr := trip.ID
	wallet, _, err := recordTransaction(
		tx, userID,
		models.WalletTxTripCharge,
		-costBreakdown.Total,
		&tripIDPtr,
		nil,
		fmt.Sprintf("Viaje #%d (%s)", trip.ID, formatDuration(durationSecs)),
	)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "could_not_charge_wallet")
		return
	}

	if err := tx.Commit().Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could_not_commit")
		return
	}

	response := map[string]any{
		"trip":               trip,
		"duration_formatted": formatDuration(durationSecs),
		"distance_formatted": formatDistance(distanceMeters),
		"minutes_charged":    costBreakdown.MinutesCharged,
		"price_per_minute":   costBreakdown.PricePerMinute,
		"subtotal":           costBreakdown.Subtotal,
		"tax_amount":         costBreakdown.TaxAmount,
		"tax_rate":           ivaRate,
		"cost":               costBreakdown.Total, // total con IVA (compatibilidad)
		"total":              costBreakdown.Total,
		"balance_after":      wallet.Balance,
		"can_rent":           wallet.CanRent(),
		"lock_sent":          lockSent,
	}

	if !lockSent {
		response["warning"] = "trip ended but lock command failed — scooter may remain unlocked"
	}

	writeJSON(w, http.StatusOK, response)
}

func GetActiveTripHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var trip models.Trip
	err := postgres.GDB.
		Preload("Scooter").
		Where("user_id = ? AND status = ?", userID, models.TripStatusActive).
		First(&trip).Error

	if err == gorm.ErrRecordNotFound {
		writeError(w, http.StatusNotFound, "no active trip")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, trip)
}

func ListTripsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Solo trips completados (excluir active y cancelled — los cancelled son
	// trips que hicieron rollback al fallar el ACK del unlock, no son viajes reales).
	var trips []models.Trip
	postgres.GDB.
		Preload("Scooter").
		Where("user_id = ? AND status = ?", userID, models.TripStatusCompleted).
		Order("started_at DESC").
		Find(&trips)

	writeJSON(w, http.StatusOK, trips)
}

func GetTripHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tripID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return
	}

	var trip models.Trip
	if err := postgres.GDB.Preload("Scooter").First(&trip, tripID).Error; err != nil {
		writeError(w, http.StatusNotFound, "trip not found")
		return
	}
	if trip.UserID != userID {
		writeError(w, http.StatusForbidden, "not your trip")
		return
	}

	writeJSON(w, http.StatusOK, trip)
}
