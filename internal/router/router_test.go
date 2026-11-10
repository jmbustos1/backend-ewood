package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Los lock/unlock directos ya no deben existir para usuarios; los flujos de
// viaje y el comando admin siguen ruteados.
func TestScooterCommandRoutes(t *testing.T) {
	t.Setenv("ADMIN_COMMAND_KEY", "")
	h := NewRouter()

	tests := []struct {
		method, path string
		wantCode     int
	}{
		{http.MethodPost, "/api/scooters/4/unlock", http.StatusNotFound},
		{http.MethodPost, "/api/scooters/4/lock", http.StatusNotFound},
		{http.MethodPost, "/api/trips", http.StatusUnauthorized},                          // ruteado, pide JWT
		{http.MethodPatch, "/api/trips/1/end", http.StatusUnauthorized},                   // ruteado, pide JWT
		{http.MethodPost, "/api/admin/scooters/4/command", http.StatusServiceUnavailable}, // ruteado, deshabilitado sin clave
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`)))
			if w.Code != tt.wantCode {
				t.Fatalf("%s %s = %d, want %d", tt.method, tt.path, w.Code, tt.wantCode)
			}
		})
	}
}
