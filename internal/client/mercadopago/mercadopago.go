// Package mercadopago implementa el cliente HTTP para la API de MercadoPago
// Checkout Pro. No usa el SDK oficial — son llamadas HTTP directas para no
// agregar dependencias y mantener el código auditable.
//
// Doc oficial:
//   - Crear preference: https://www.mercadopago.cl/developers/es/reference/preferences/_checkout_preferences/post
//   - Obtener payment:  https://www.mercadopago.cl/developers/es/reference/payments/_payments_id/get
package mercadopago

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const baseURL = "https://api.mercadopago.com"

// ── Tipos para crear una preference ─────────────────────────────────────────

// PreferenceItem es un item del carrito en la preference. Para recargas usamos
// un único item con el monto de la recarga.
type PreferenceItem struct {
	Title       string  `json:"title"`
	Quantity    int     `json:"quantity"`
	CurrencyID  string  `json:"currency_id"`
	UnitPrice   float64 `json:"unit_price"`
	Description string  `json:"description,omitempty"`
}

// BackURLs son las URLs a las que MP redirige al usuario al terminar el flow.
type BackURLs struct {
	Success string `json:"success"`
	Failure string `json:"failure"`
	Pending string `json:"pending"`
}

// PreferenceRequest es el body que se envía a POST /checkout/preferences.
type PreferenceRequest struct {
	Items             []PreferenceItem `json:"items"`
	ExternalReference string           `json:"external_reference,omitempty"`
	BackURLs          *BackURLs        `json:"back_urls,omitempty"`
	AutoReturn        string           `json:"auto_return,omitempty"`     // "approved" para volver auto
	NotificationURL   string           `json:"notification_url,omitempty"` // webhook
	StatementDescriptor string         `json:"statement_descriptor,omitempty"`
}

// PreferenceResponse es lo que devuelve MP al crear la preference.
// init_point es la URL del checkout que se abre en la WebView.
type PreferenceResponse struct {
	ID                string `json:"id"`
	InitPoint         string `json:"init_point"`
	SandboxInitPoint  string `json:"sandbox_init_point"`
	ExternalReference string `json:"external_reference"`
	DateCreated       string `json:"date_created"`
}

// ── Tipos para consultar un payment ─────────────────────────────────────────

// PaymentResponse es la respuesta de GET /v1/payments/{id}.
// Sólo mapeamos los campos que nos interesan; MP devuelve muchos más.
type PaymentResponse struct {
	ID                int     `json:"id"`
	Status            string  `json:"status"`           // approved | pending | rejected | in_process | refunded
	StatusDetail      string  `json:"status_detail"`
	TransactionAmount float64 `json:"transaction_amount"`
	CurrencyID        string  `json:"currency_id"`
	ExternalReference string  `json:"external_reference"`
	DateApproved      string  `json:"date_approved,omitempty"`
	PaymentMethodID   string  `json:"payment_method_id,omitempty"`
	PaymentTypeID     string  `json:"payment_type_id,omitempty"`
}

// ── Cliente ─────────────────────────────────────────────────────────────────

// Client encapsula la configuración para llamar a la API de MP.
type Client struct {
	AccessToken string
	HTTPClient  *http.Client
}

// New construye un cliente con el access token desde la env MP_ACCESS_TOKEN.
// Devuelve error si la env no está seteada.
func New() (*Client, error) {
	token := strings.TrimSpace(os.Getenv("MP_ACCESS_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("MP_ACCESS_TOKEN no está configurada")
	}
	return &Client{
		AccessToken: token,
		HTTPClient:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// CreatePreference llama a POST /checkout/preferences y devuelve la URL de checkout.
func (c *Client) CreatePreference(req *PreferenceRequest) (*PreferenceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error serializando preference: %w", err)
	}

	httpReq, err := http.NewRequest("POST", baseURL+"/checkout/preferences", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a MP: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("MP CreatePreference fallo (status=%d): %s", resp.StatusCode, string(respBody))
	}

	var result PreferenceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta MP: %w", err)
	}
	return &result, nil
}

// GetPayment llama a GET /v1/payments/{id} para verificar el estado real del
// pago. Se usa en el webhook: nunca confiar solo en la notificación.
func (c *Client) GetPayment(paymentID string) (*PaymentResponse, error) {
	httpReq, err := http.NewRequest("GET", baseURL+"/v1/payments/"+paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a MP: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MP GetPayment fallo (status=%d): %s", resp.StatusCode, string(respBody))
	}

	var result PaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta MP: %w", err)
	}
	return &result, nil
}

// IsApproved retorna true si el payment está confirmado (la plata efectivamente
// llegó). Cualquier otro estado (pending, rejected, etc.) NO debe acreditar saldo.
func (p *PaymentResponse) IsApproved() bool {
	return p != nil && p.Status == "approved"
}

// ── Merchant Orders ─────────────────────────────────────────────────────────

// MerchantOrderPayment es un payment listado dentro de una merchant_order.
// Tiene menos campos que GetPayment; usar GetPayment para detalles completos.
type MerchantOrderPayment struct {
	ID                int     `json:"id"`
	Status            string  `json:"status"`
	TransactionAmount float64 `json:"transaction_amount"`
	DateApproved      string  `json:"date_approved,omitempty"`
}

// MerchantOrderResponse es la respuesta de GET /merchant_orders/{id}.
type MerchantOrderResponse struct {
	ID                int                    `json:"id"`
	Status            string                 `json:"status"`            // opened | closed | expired
	OrderStatus       string                 `json:"order_status"`       // payment_required | paid | partially_paid
	PreferenceID      string                 `json:"preference_id"`
	ExternalReference string                 `json:"external_reference"`
	Payments          []MerchantOrderPayment `json:"payments"`
	TotalAmount       float64                `json:"total_amount"`
	PaidAmount        float64                `json:"paid_amount"`
}

// GetMerchantOrder consulta una merchant order por ID. Se usa cuando MP
// notifica con topic=merchant_order — la orden contiene la lista de payments
// asociados a la preference.
func (c *Client) GetMerchantOrder(id string) (*MerchantOrderResponse, error) {
	httpReq, err := http.NewRequest("GET", baseURL+"/merchant_orders/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a MP: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MP GetMerchantOrder fallo (status=%d): %s", resp.StatusCode, string(respBody))
	}

	var result MerchantOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta MP: %w", err)
	}
	return &result, nil
}
