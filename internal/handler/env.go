package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// EnvHandler muestra las variables de entorno configuradas (sin mostrar contraseñas)
func EnvHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	env := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"environment": map[string]interface{}{
			"ENV":  os.Getenv("ENV"),
			"PORT": os.Getenv("PORT"),
		},
		"postgresql": map[string]interface{}{
			"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
			"POSTGRES_PORT":     getEnvWithDefault("POSTGRES_PORT", "5432"),
			"POSTGRES_USER":     maskIfNotEmpty(os.Getenv("POSTGRES_USER")),
			"POSTGRES_PASSWORD": maskIfNotEmpty(os.Getenv("POSTGRES_PASSWORD")),
			"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
			"POSTGRES_SSLMODE":  getPostgresSSLMode(),
			"configured":        os.Getenv("POSTGRES_HOST") != "",
		},
		"mongodb": map[string]interface{}{
			"MONGODB_URI":      maskURI(os.Getenv("MONGODB_URI")),
			"MONGODB_DATABASE": os.Getenv("MONGODB_DATABASE"),
			"configured":       os.Getenv("MONGODB_URI") != "",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(env)
}

// getEnvWithDefault retorna el valor de una variable de entorno o un valor por defecto
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// maskIfNotEmpty enmascara el valor si no está vacío
func maskIfNotEmpty(value string) string {
	if value == "" {
		return "<not_set>"
	}
	return "***"
}

// getPostgresSSLMode retorna el SSLMODE según el entorno
func getPostgresSSLMode() string {
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode != "" {
		return sslmode
	}
	// Por defecto, disable para local, require para producción
	env := os.Getenv("ENV")
	if env == "production" {
		return "require (default)"
	}
	return "disable (default)"
}

// maskURI enmascara las credenciales en una URI de conexión
func maskURI(uri string) string {
	if uri == "" {
		return "<not_set>"
	}
	// Si contiene @, probablemente tiene credenciales
	// Formato típico: protocol://user:password@host
	if strings.Contains(uri, "@") {
		// Buscar el @ y reemplazar todo antes del @ (excepto el protocolo)
		parts := strings.Split(uri, "@")
		if len(parts) == 2 {
			// Separar protocolo de credenciales
			if idx := strings.Index(parts[0], "://"); idx > 0 {
				protocol := parts[0][:idx+3]
				return protocol + "***:***@" + parts[1]
			}
		}
	}
	// Si no podemos parsear, solo mostrar que está configurado
	return "<configured>"
}

