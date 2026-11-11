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

// fakeACK responde el próximo comando con el status y result dados.
func fakeACK(t *testing.T, m *TCPConnectionManager, lines chan map[string]any, wantCmd, status string, result any) {
	t.Helper()
	go func() {
		msg := <-lines
		if msg["command"] != wantCmd {
			t.Errorf("command = %v, want %v", msg["command"], wantCmd)
		}
		ack := map[string]interface{}{"message_id": msg["message_id"], "status": status}
		if result != nil {
			ack["result"] = result
		}
		m.ProcessACK(4, ack)
	}()
}

func TestAdminCommandReturnsACKResult(t *testing.T) {
	m := newTestManager(time.Second)
	withManager(t, m)
	t.Setenv("ADMIN_COMMAND_KEY", testAdminKey)
	_, lines := fakeScooter(t, m, 4)

	fakeACK(t, m, lines, "gps_status", "ok", map[string]any{"on": 1, "fix": 3, "sv": []int{8, 4, 2}})
	w := httptest.NewRecorder()
	AdminScooterCommandHandler(w, adminRequest("4", testAdminKey, `{"command":"gps_status"}`))
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, `"sv":[8,4,2]`) {
		t.Fatalf("got %d %s", w.Code, body)
	}

	fakeACK(t, m, lines, "gps_recycle", "error", map[string]any{"error": "gps_busy"})
	w = httptest.NewRecorder()
	AdminScooterCommandHandler(w, adminRequest("4", testAdminKey, `{"command":"gps_recycle"}`))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "gps_busy") {
		t.Fatalf("ACK con error: got %d %s; want 502 gps_busy", w.Code, w.Body.String())
	}
}

func TestAdminDisruptiveCommandsRespectTrips(t *testing.T) {
	m := newTestManager(time.Second)
	withManager(t, m)
	t.Setenv("ADMIN_COMMAND_KEY", testAdminKey)
	_, lines := fakeScooter(t, m, 4)

	prev := scooterInTrip
	t.Cleanup(func() { scooterInTrip = prev })
	scooterInTrip = func(uint) (bool, error) { return true, nil }

	for _, c := range []string{"reboot", "gps_cold"} {
		w := httptest.NewRecorder()
		AdminScooterCommandHandler(w, adminRequest("4", testAdminKey, `{"command":"`+c+`"}`))
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "scooter_in_trip") {
			t.Fatalf("%s en viaje: got %d %s; want 409", c, w.Code, w.Body.String())
		}
	}

	// gps_status no es disruptivo: no consulta viajes
	scooterInTrip = func(uint) (bool, error) { t.Fatal("no debía consultar viajes"); return false, nil }
	fakeACK(t, m, lines, "gps_status", "ok", nil)
	w := httptest.NewRecorder()
	AdminScooterCommandHandler(w, adminRequest("4", testAdminKey, `{"command":"gps_status"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("gps_status: got %d %s", w.Code, w.Body.String())
	}

	// force:true salta la verificación
	fakeACK(t, m, lines, "reboot", "ok", nil)
	w = httptest.NewRecorder()
	AdminScooterCommandHandler(w, adminRequest("4", testAdminKey, `{"command":"reboot","force":true}`))
	if w.Code != http.StatusOK {
		t.Fatalf("reboot force: got %d %s", w.Code, w.Body.String())
	}
}
