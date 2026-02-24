package config

import (
	"os"
)

// Config contiene la configuración de la aplicación
type Config struct {
	Env      string
	Port     string
	Database DatabaseConfig
}

// DatabaseConfig contiene la configuración de las bases de datos
type DatabaseConfig struct {
	PostgreSQL PostgreSQLConfig
	MongoDB    MongoDBConfig
}

// PostgreSQLConfig contiene la configuración de PostgreSQL
type PostgreSQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// MongoDBConfig contiene la configuración de MongoDB
type MongoDBConfig struct {
	URI      string
	Database string
}

// LoadConfig carga la configuración desde variables de entorno
func LoadConfig() *Config {
	return &Config{
		Env:  getEnv("ENV", "development"),
		Port: getEnv("PORT", ":8080"),
		Database: DatabaseConfig{
			PostgreSQL: PostgreSQLConfig{
				Host:     os.Getenv("POSTGRES_HOST"),
				Port:     getEnv("POSTGRES_PORT", "5432"),
				User:     os.Getenv("POSTGRES_USER"),
				Password: os.Getenv("POSTGRES_PASSWORD"),
				DBName:   os.Getenv("POSTGRES_DB"),
				SSLMode:  getPostgresSSLMode(),
			},
			MongoDB: MongoDBConfig{
				URI:      os.Getenv("MONGODB_URI"),
				Database: os.Getenv("MONGODB_DATABASE"),
			},
		},
	}
}

// getEnv retorna el valor de una variable de entorno o un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
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
		return "require"
	}
	return "disable"
}

