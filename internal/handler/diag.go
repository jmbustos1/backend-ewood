package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
)

// Diagnóstico del firmware: bloque "d" de la telemetría TCP.
// Significado de cada campo: esp32ewood/docs/DIAGNOSTICO.md.

// maxDiagBytes limita lo que se guarda por scooter (el firmware envía ~160).
const maxDiagBytes = 1024

// diagKeyFields son los campos cuyo cambio vale una línea de log.
var diagKeyFields = []string{"fw", "rr", "ps", "tr", "hb", "gr", "gv", "bv"}

// parseDiag valida que el bloque sea un objeto JSON razonable.
func parseDiag(raw json.RawMessage) (map[string]any, bool) {
	if len(raw) == 0 || len(raw) > maxDiagBytes {
		return nil, false
	}
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil || d == nil {
		return nil, false
	}
	return d, true
}

// diffDiag describe los cambios relevantes entre el bloque anterior y el
// actual. Retorna nil si no hay nada que registrar.
func diffDiag(prev, cur map[string]any) []string {
	if prev == nil {
		return []string{"primer diagnóstico"}
	}

	var changes []string
	if pu, ok := asNumber(prev["up"]); ok {
		if cu, ok := asNumber(cur["up"]); ok && cu < pu {
			changes = append(changes, fmt.Sprintf("REINICIO (up %v→%v, rr=%v)", pu, cu, cur["rr"]))
		}
	}
	for _, k := range diagKeyFields {
		if fmt.Sprint(prev[k]) != fmt.Sprint(cur[k]) {
			changes = append(changes, fmt.Sprintf("%s %v→%v", k, prev[k], cur[k]))
		}
	}
	return changes
}

// logDiagChanges registra en el log los cambios relevantes de diagnóstico.
func logDiagChanges(scooterID uint, prevRaw *string, cur map[string]any) {
	var prev map[string]any
	if prevRaw != nil {
		_ = json.Unmarshal([]byte(*prevRaw), &prev)
	}
	changes := diffDiag(prev, cur)
	if len(changes) == 0 {
		return
	}
	log.Printf("📟 Scooter %d diag: %s | %s", scooterID, strings.Join(changes, ", "), formatDiag(cur))
}

// formatDiag imprime el bloque con claves ordenadas, para logs legibles.
func formatDiag(d map[string]any) string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, d[k]))
	}
	return strings.Join(parts, " ")
}

func asNumber(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}
