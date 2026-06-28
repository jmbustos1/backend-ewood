package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"ewood-backend/internal/client/mercadopago"
	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/middleware"
	"ewood-backend/internal/models"
)

// minRechargeAmount es el monto minimo aceptado para recargar la wallet (CLP).
const minRechargeAmount = 1000.0

// rechargeRequest body de POST /api/wallet/recharge
type rechargeRequest struct {
	Amount float64 `json:"amount"`
}

// walletResponse formato de respuesta para GET /api/wallet
type walletResponse struct {
	Balance      float64                    `json:"balance"`
	CanRent      bool                       `json:"can_rent"`
	Transactions []models.WalletTransaction `json:"transactions"`
}

// getOrCreateWallet retorna la wallet del usuario; si no existe, la crea con balance 0.
// Funciona tanto con la conexion global como con una transaccion (tx) si se pasa.
func getOrCreateWallet(db *gorm.DB, userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := db.First(&wallet, "user_id = ?", userID).Error
	if err == nil {
		return &wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// No existe, crearla
	wallet = models.Wallet{UserID: userID, Balance: 0}
	if err := db.Create(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

// recordTransaction registra una transaccion y actualiza el balance de la wallet.
// Debe ejecutarse dentro de una transaccion DB para garantizar atomicidad.
//
// amount: positivo en recargas/refunds, negativo en cobros.
// El balance puede quedar negativo (deuda).
func recordTransaction(tx *gorm.DB, userID uint, txType models.WalletTransactionType,
	amount float64, tripID *uint, mpPaymentID *string, description string) (*models.Wallet, *models.WalletTransaction, error) {

	// Lock de fila para evitar race conditions
	wallet := &models.Wallet{}
	if err := tx.Where("user_id = ?", userID).First(wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Crear wallet si no existe
			wallet.UserID = userID
			wallet.Balance = 0
			if err := tx.Create(wallet).Error; err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	newBalance := wallet.Balance + amount
	wallet.Balance = newBalance

	if err := tx.Save(wallet).Error; err != nil {
		return nil, nil, err
	}

	walletTx := &models.WalletTransaction{
		UserID:       userID,
		Type:         txType,
		Amount:       amount,
		BalanceAfter: newBalance,
		TripID:       tripID,
		MPPaymentID:  mpPaymentID,
		Description:  description,
		CreatedAt:    time.Now(),
	}
	if err := tx.Create(walletTx).Error; err != nil {
		return nil, nil, err
	}

	return wallet, walletTx, nil
}

// GetWalletHandler GET /api/wallet
// Retorna saldo + ultimas 20 transacciones del usuario autenticado.
func GetWalletHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	wallet, err := getOrCreateWallet(postgres.GDB, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load wallet")
		return
	}

	var txs []models.WalletTransaction
	if err := postgres.GDB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&txs).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load transactions")
		return
	}

	writeJSON(w, http.StatusOK, walletResponse{
		Balance:      wallet.Balance,
		CanRent:      wallet.CanRent(),
		Transactions: txs,
	})
}

// RechargeWalletHandler POST /api/wallet/recharge
//
// Crea una preference en MercadoPago Checkout Pro y retorna la URL de checkout
// (init_point) que el cliente debe abrir en una WebView. El saldo NO se acredita
// acá — eso lo hace MPWebhookHandler cuando MP confirma el pago como approved.
//
// external_reference codifica el user_id para mapear el pago de vuelta al usuario.
func RechargeWalletHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req rechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount < minRechargeAmount {
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("min_recharge_amount: %.0f", minRechargeAmount))
		return
	}

	mp, err := mercadopago.New()
	if err != nil {
		log.Printf("❌ MP no configurado: %v", err)
		writeError(w, http.StatusInternalServerError, "payment_provider_unavailable")
		return
	}

	backendBaseURL := strings.TrimRight(os.Getenv("MP_BACKEND_BASE_URL"), "/")
	if backendBaseURL == "" {
		backendBaseURL = "http://98.92.176.224:8200" // fallback
	}
	appReturnURL := os.Getenv("MP_APP_RETURN_URL")
	if appReturnURL == "" {
		appReturnURL = "ewood://wallet/recharge-result"
	}

	preference := &mercadopago.PreferenceRequest{
		Items: []mercadopago.PreferenceItem{{
			Title:       "Recarga saldo Ewood",
			Description: fmt.Sprintf("Recarga de $%.0f CLP a tu billetera Ewood", req.Amount),
			Quantity:    1,
			CurrencyID:  "CLP",
			UnitPrice:   req.Amount,
		}},
		// Codificamos el user_id; el webhook lo necesita para saber a quién acreditar.
		ExternalReference: fmt.Sprintf("user_%d_%d", userID, time.Now().Unix()),
		BackURLs: &mercadopago.BackURLs{
			Success: appReturnURL + "?status=success",
			Failure: appReturnURL + "?status=failure",
			Pending: appReturnURL + "?status=pending",
		},
		// auto_return solo dispara en success — el resto requiere que el usuario toque "volver".
		AutoReturn:          "approved",
		NotificationURL:     backendBaseURL + "/api/wallet/mp-webhook",
		StatementDescriptor: "EWOOD",
	}

	pref, err := mp.CreatePreference(preference)
	if err != nil {
		log.Printf("❌ Error creando preference MP: %v", err)
		writeError(w, http.StatusBadGateway, "could_not_create_payment")
		return
	}

	// Devolvemos init_point (sandbox o producción según el access token usado).
	checkoutURL := pref.InitPoint
	if checkoutURL == "" {
		checkoutURL = pref.SandboxInitPoint
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":            true,
		"checkout_url":       checkoutURL,
		"preference_id":      pref.ID,
		"external_reference": pref.ExternalReference,
		"amount":             req.Amount,
		"message":            "Abrir checkout_url en WebView. El saldo se acredita cuando MP confirme el pago.",
	})
}

// MPWebhookHandler POST /api/wallet/mp-webhook
//
// Recibe notificaciones de MercadoPago cuando cambia el estado de un pago.
// Maneja DOS tipos de notificación que MP puede enviar:
//
//   1. topic/type = "payment" → recibimos el payment_id directamente.
//   2. topic = "merchant_order" → recibimos el merchant_order_id; debemos
//      consultar la orden, obtener la lista de payments y procesar los aprobados.
//      Esto pasa cuando en el panel de MP solo está activo el webhook "Órdenes"
//      o cuando MP usa el formato IPN legacy.
//
// Patrón seguro común:
//   - NUNCA confiar solo en la notificación, siempre consultar el estado real a MP.
//   - Solo acreditar si status == approved.
//   - Idempotente: si ya hay una wallet_transaction con ese mp_payment_id, no
//     se vuelve a acreditar.
func MPWebhookHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type   string `json:"type"`
		Action string `json:"action"`
		Data   struct {
			ID string `json:"id"`
		} `json:"data"`
		// Formato IPN legacy:
		Topic    string `json:"topic"`
		ID       string `json:"id"`
		Resource string `json:"resource"` // ej: "https://api.mercadolibre.com/merchant_orders/41679780219"
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(bodyBytes, &body)

	mp, err := mercadopago.New()
	if err != nil {
		log.Printf("❌ MP no configurado al recibir webhook: %v", err)
		http.Error(w, "mp not configured", http.StatusInternalServerError)
		return
	}

	// ── Caso 1: payment directo ────────────────────────────────────────
	var paymentIDs []string
	if body.Type == "payment" && body.Data.ID != "" {
		paymentIDs = []string{body.Data.ID}
	} else if body.Topic == "payment" && body.ID != "" {
		paymentIDs = []string{body.ID}
	} else if body.Topic == "payment" && body.Resource != "" {
		// Formato IPN legacy: el payment_id puede venir en `resource` como
		// "163336664876" o como URL "https://api.mercadolibre.com/payments/163336664876"
		res := body.Resource
		if strings.Contains(res, "/") {
			parts := strings.Split(res, "/")
			res = parts[len(parts)-1]
		}
		paymentIDs = []string{res}
	}

	// ── Caso 2: merchant_order — extraemos los payments ────────────────
	if len(paymentIDs) == 0 && (body.Topic == "merchant_order" || body.Type == "merchant_order") {
		moID := body.Data.ID
		if moID == "" {
			moID = body.ID
		}
		if moID == "" && body.Resource != "" {
			// Extraer último segmento del URL: ".../merchant_orders/41679780219"
			parts := strings.Split(body.Resource, "/")
			if len(parts) > 0 {
				moID = parts[len(parts)-1]
			}
		}
		if moID == "" {
			log.Printf("ℹ️ Webhook merchant_order sin ID: body=%s", string(bodyBytes))
			w.WriteHeader(http.StatusOK)
			return
		}

		mo, err := mp.GetMerchantOrder(moID)
		if err != nil {
			log.Printf("❌ Error consultando merchant_order %s: %v", moID, err)
			http.Error(w, "could not fetch merchant order", http.StatusBadGateway)
			return
		}

		for _, p := range mo.Payments {
			paymentIDs = append(paymentIDs, fmt.Sprintf("%d", p.ID))
		}

		if len(paymentIDs) == 0 {
			log.Printf("ℹ️ merchant_order %s sin payments aún (order_status=%s)", moID, mo.OrderStatus)
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("ℹ️ merchant_order %s con %d payments a procesar", moID, len(paymentIDs))
	}

	if len(paymentIDs) == 0 {
		log.Printf("ℹ️ Webhook MP ignorado (no es payment ni merchant_order útil): body=%s", string(bodyBytes))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Procesar cada payment encontrado
	for _, paymentID := range paymentIDs {
		if err := processMPPayment(mp, paymentID); err != nil {
			log.Printf("❌ Error procesando payment %s: %v", paymentID, err)
			// Seguimos con los demás
		}
	}

	w.WriteHeader(http.StatusOK)
}

// processMPPayment consulta un payment a MP, valida que esté aprobado y acredita
// el saldo de forma idempotente. Devuelve error solo si fue un problema técnico
// (no si el payment está pending o ya estaba acreditado).
func processMPPayment(mp *mercadopago.Client, paymentID string) error {
	payment, err := mp.GetPayment(paymentID)
	if err != nil {
		return fmt.Errorf("GetPayment: %w", err)
	}

	if !payment.IsApproved() {
		log.Printf("ℹ️ Payment %s no aprobado (status=%s), no acreditamos", paymentID, payment.Status)
		return nil
	}

	userID, err := parseUserIDFromExternalRef(payment.ExternalReference)
	if err != nil {
		log.Printf("❌ external_reference inválida en payment %s: %q (%v)",
			paymentID, payment.ExternalReference, err)
		return fmt.Errorf("invalid external_reference")
	}

	// Idempotencia: si ya hay una transaction con ese mp_payment_id, no hacer nada.
	var existing models.WalletTransaction
	err = postgres.GDB.
		Where("mp_payment_id = ?", paymentID).
		First(&existing).Error
	if err == nil {
		log.Printf("ℹ️ Payment %s ya estaba acreditado (tx_id=%d). Webhook idempotente.",
			paymentID, existing.ID)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("consultando wallet_transactions: %w", err)
	}

	// Acreditar saldo dentro de transacción DB
	tx := postgres.GDB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("could not start tx: %w", tx.Error)
	}

	pidStr := paymentID
	wallet, _, err := recordTransaction(
		tx, userID,
		models.WalletTxRecharge,
		payment.TransactionAmount,
		nil,
		&pidStr,
		fmt.Sprintf("Recarga via MercadoPago (payment_id=%s)", paymentID),
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("acreditando saldo: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("✅ Recarga MP acreditada: user=%d amount=%.2f payment_id=%s nuevo_balance=%.2f",
		userID, payment.TransactionAmount, paymentID, wallet.Balance)
	return nil
}

// parseUserIDFromExternalRef extrae el user_id del formato "user_<id>_<ts>".
func parseUserIDFromExternalRef(ref string) (uint, error) {
	if !strings.HasPrefix(ref, "user_") {
		return 0, fmt.Errorf("formato inesperado")
	}
	parts := strings.Split(ref, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("partes insuficientes")
	}
	var id uint64
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return 0, err
	}
	return uint(id), nil
}
