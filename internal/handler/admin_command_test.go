package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

const testAdminKey = "test-admin-key-0123456789abcdefghij" // >= minAdminKeyLen

// withManager reemplaza el tcpManager global durante el test.
func withManager(t *testing.T, m *TCPConnectionManager) {
	t.Helper()
	prev := tcpManager
	tcpManager = m
	t.Cleanup(func() { tcpManager = prev })
}

func adminRequest(id, key, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/admin/scooters/"+id+"/command", strings.NewReader(body))
	if key != "" {
		r.Header.Set("X-Admin-Key", key)
	}
	return mux.SetURLVars(r, map[string]string{"id": id})
}

func TestAdminCommandAuthAndValidation(t *testing.T) {
	withManager(t, newTestManager(time.Second))

	tests := []struct {
		name     string
		envKey   string
		key      string
		body     string
		wantCode int
		wantBody string
	}{
		{"deshabilitado sin clave", "", testAdminKey, `{"command":"lock"}`, http.StatusServiceUnavailable, "admin_commands_disabled"},
		{"deshabilitado con clave corta", "corta", "corta", `{"command":"lock"}`, http.StatusServiceUnavailable, "admin_commands_disabled"},
		{"sin header", testAdminKey, "", `{"command":"lock"}`, http.StatusUnauthorized, "unauthorized"},
		{"clave incorrecta", testAdminKey, testAdminKey + "x", `{"command":"lock"}`, http.StatusUnauthorized, "unauthorized"},
		{"comando no permitido", testAdminKey, testAdminKey, `{"command":"AT+CRESET"}`, http.StatusBadRequest, "command_not_allowed"},
		{"body inválido", testAdminKey, testAdminKey, `nope`, http.StatusBadRequest, "invalid_request_body"},
		{"scooter desconectado", testAdminKey, testAdminKey, `{"command":"lock"}`, http.StatusServiceUnavailable, "scooter_not_connected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADMIN_COMMAND_KEY", tt.envKey)
			w := httptest.NewRecorder()
			AdminScooterCommandHandler(w, adminRequest("4", tt.key, tt.body))
			if w.Code != tt.wantCode || !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Fatalf("got %d %s; want %d %s", w.Code, w.Body.String(), tt.wantCode, tt.wantBody)
			}
		})
	}
}

func TestAdminCommandSendsAndReportsACK(t *testing.T) {
	m := newTestManager(time.Second)
	withManager(t, m)
	t.Setenv("ADMIN_COMMAND_KEY", testAdminKey)
	_, lines := fakeScooter(t, m, 4)

	go func() {
		msg := <-lines
		if msg["command"] != "lock" {
			t.Errorf("command = %v, want lock", msg["command"])
		}
		m.ProcessACK(4, map[string]interface{}{"message_id": msg["message_id"]})
	}()

	w := httptest.NewRecorder()
	AdminScooterCommandHandler(w, adminRequest("4", testAdminKey, `{"command":"lock"}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ack":true`) {
		t.Fatalf("got %d %s; want 200 ack:true", w.Code, w.Body.String())
	}
}

// El WebSocket ya no debe reenviar comandos de clientes a los scooters.
func TestWebSocketSendToScooterIsIgnored(t *testing.T) {
	m := newTestManager(time.Second)
	withManager(t, m)
	_, lines := fakeScooter(t, m, 4)

	msg := `{"action":"send_to_scooter","scooter_id":4,"message":{"command":"unlock"},"_client_id":"client_x"}`
	if err := HandleFrontendMessage([]byte(msg)); err != nil {
		t.Fatalf("err = %v", err)
	}

	select {
	case got := <-lines:
		t.Fatalf("el scooter recibió un comando desde el WebSocket: %v", got)
	case <-time.After(200 * time.Millisecond):
	}
}
