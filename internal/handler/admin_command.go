package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/models"

	"github.com/gorilla/mux"
)

// minAdminKeyLen evita habilitar el endpoint con una clave débil por error.
const minAdminKeyLen = 32

// adminCommands son los únicos comandos aceptados por el endpoint de
// emergencia. Nunca aceptar AT libres: un comando mal puesto puede dejar el
// módem inaccesible. Al agregar comandos de mantenimiento (gps_status,
// gps_recycle, ...) el nombre no debe contener "lock": el firmware tiene una
// ruta de texto plano que busca esa palabra dentro del mensaje.
// Referencia: docs/COMANDOS_EMERGENCIA.md.
var adminCommands = map[string]bool{
	"lock":        true,
	"unlock":      true,
	"gps_status":  true, // estado del motor GNSS (requiere firmware >= comandos GPS)
	"gps_recycle": true, // apaga y prende el GNSS (asíncrono en el firmware)
	"gps_cold":    true, // como gps_recycle + borra efemérides (arranque en frío)
	"reboot":      true, // esp_restart() tras enviar el ACK
}

// disruptiveCommands se rechazan si el scooter está en un viaje activo, salvo
// que el request traiga "force": true.
var disruptiveCommands = map[string]bool{
	"reboot":   true,
	"gps_cold": true,
}

type adminCommandRequest struct {
	Command string `json:"command"`
	Force   bool   `json:"force,omitempty"`
}

// scooterInTrip consulta si el scooter está en uso. Variable para tests.
var scooterInTrip = func(id uint) (bool, error) {
	var s models.Scooter
	if err := postgres.GDB.Select("id", "status").First(&s, id).Error; err != nil {
		return false, err
	}
	return s.IsInUse(), nil
}

// AdminScooterCommandHandler envía un comando de la lista permitida a un
// scooter, fuera del flujo de viajes. Solo para operación (emergencias).
//
//	POST /api/admin/scooters/{id}/command
//	X-Admin-Key: <ADMIN_COMMAND_KEY>
//	{"command": "lock"}
//
// No cambia el estado del scooter ni de viajes en la base: solo envía el
// comando y espera el ACK. Si ADMIN_COMMAND_KEY no está configurada (o es
// demasiado corta), el endpoint queda deshabilitado.
func AdminScooterCommandHandler(w http.ResponseWriter, r *http.Request) {
	origin := requestOrigin(r)

	expected := os.Getenv("ADMIN_COMMAND_KEY")
	if len(expected) < minAdminKeyLen {
		writeError(w, http.StatusServiceUnavailable, "admin_commands_disabled")
		return
	}

	got := r.Header.Get("X-Admin-Key")
	if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		log.Printf("🚫 Comando admin rechazado: clave inválida (origen %s)", origin)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "invalid_scooter_id")
		return
	}

	var req adminCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_body")
		return
	}
	if !adminCommands[req.Command] {
		writeError(w, http.StatusBadRequest, "command_not_allowed")
		return
	}

	if disruptiveCommands[req.Command] && !req.Force {
		inTrip, err := scooterInTrip(uint(id))
		if err != nil {
			writeError(w, http.StatusNotFound, "scooter_not_found")
			return
		}
		if inTrip {
			writeError(w, http.StatusConflict, "scooter_in_trip")
			return
		}
	}

	out, cmdErr := sendScooterCommandWithResult(uint(id), req.Command)
	log.Printf("🛠️  Comando admin %q a scooter %d (origen %s, force=%v): ack=%v status=%q result=%s message_id=%d err=%v",
		req.Command, id, origin, req.Force, out.Acked, out.Status, string(out.Result), out.MessageID, cmdErr)

	if cmdErr != nil {
		writeError(w, http.StatusServiceUnavailable, cmdErr.Error())
		return
	}

	status := http.StatusOK
	switch {
	case !out.Acked:
		status = http.StatusGatewayTimeout
	case out.Status == "error":
		status = http.StatusBadGateway // el firmware respondió, pero con error
	}

	resp := map[string]any{
		"scooter_id": id,
		"command":    req.Command,
		"ack":        out.Acked,
		"message_id": out.MessageID,
	}
	if out.Status != "" {
		resp["status"] = out.Status
	}
	if len(out.Result) > 0 {
		resp["result"] = out.Result
	}
	writeJSON(w, status, resp)
}

// requestOrigin retorna la IP del cliente para logs. Detrás de Caddy, la IP
// real viene en X-Forwarded-For; es solo informativa (puede falsificarse).
func requestOrigin(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd + " via " + r.RemoteAddr
	}
	return r.RemoteAddr
}
