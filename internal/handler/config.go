package handler

import (
	"encoding/json"
	"net/http"
)

// ConfigHandler expone configuración pública consumida por la app mobile
// (tarifa por minuto, IVA, versión mínima soportada). Sin JWT — cacheable.
// La fuente de verdad del precio es getPricePerMinute() (ver trip.go), que
// lee PRICE_PER_MINUTE del env y cae al default si no está seteada.
func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")

	resp := map[string]interface{}{
		"pricing": map[string]interface{}{
			"price_per_minute_clp": getPricePerMinute(),
			"tax_rate":             ivaRate,
			"currency":             "CLP",
		},
		"app": map[string]interface{}{
			"min_supported_version": "1.0.0",
		},
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
