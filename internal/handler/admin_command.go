package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

// minAdminKeyLen evita habilitar el endpoint con una clave débil por error.
const minAdminKeyLen = 32

// adminCommands son los únicos comandos aceptados por el endpoint de
// emergencia. Nunca aceptar AT libres: un comando mal puesto puede dejar el
// módem inaccesible. Al agregar comandos de mantenimiento (gps_status,
// gps_recycle, ...) el nombre no debe contener "lock": el firmware tiene una
// ruta de texto plano que busca esa palabra dentro del mensaje.
var adminCommands = map[string]bool{
	"lock":   true,
	"unlock": true,
}

type adminCommandRequest struct {
	Command string `json:"command"`
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

	ack, messageID, cmdErr := sendScooterCommand(uint(id), req.Command)
	log.Printf("🛠️  Comando admin %q a scooter %d (origen %s): ack=%v message_id=%d err=%v",
		req.Command, id, origin, ack, messageID, cmdErr)

	if cmdErr != nil {
		writeError(w, http.StatusServiceUnavailable, cmdErr.Error())
		return
	}

	status := http.StatusOK
	if !ack {
		status = http.StatusGatewayTimeout
	}
	writeJSON(w, status, map[string]any{
		"scooter_id": id,
		"command":    req.Command,
		"ack":        ack,
		"message_id": messageID,
	})
}

// requestOrigin retorna la IP del cliente para logs. Detrás de Caddy, la IP
// real viene en X-Forwarded-For; es solo informativa (puede falsificarse).
func requestOrigin(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd + " via " + r.RemoteAddr
	}
	return r.RemoteAddr
}
