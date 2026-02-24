package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"ewood-backend/config"
	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/migration"
)

func main() {
	var (
		command        = flag.String("command", "up", "Comando a ejecutar: up, down, version")
		migrationsPath = flag.String("path", "./migrations", "Ruta a las migraciones")
	)
	flag.Parse()

	// Cargar configuración
	cfg := config.LoadConfig()

	// Validar que PostgreSQL esté configurado
	if cfg.Database.PostgreSQL.Host == "" {
		log.Fatal("❌ POSTGRES_HOST no está configurado")
	}

	// Conectar a PostgreSQL
	if err := postgres.Connect(
		cfg.Database.PostgreSQL.Host,
		cfg.Database.PostgreSQL.Port,
		cfg.Database.PostgreSQL.User,
		cfg.Database.PostgreSQL.Password,
		cfg.Database.PostgreSQL.DBName,
		cfg.Database.PostgreSQL.SSLMode,
	); err != nil {
		log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
	}
	defer postgres.Close()

	// Obtener la conexión SQL nativa para las migraciones
	db := postgres.DB

	switch *command {
	case "up":
		log.Println("🔄 Ejecutando migraciones...")
		if err := migration.RunMigrations(db, *migrationsPath); err != nil {
			log.Fatalf("❌ Error ejecutando migraciones: %v", err)
		}
		log.Println("✅ Migraciones completadas")

	case "down":
		log.Println("🔄 Revirtiendo última migración...")
		if err := migration.RollbackLastMigration(db, *migrationsPath); err != nil {
			log.Fatalf("❌ Error revirtiendo migración: %v", err)
		}
		log.Println("✅ Migración revertida")

	case "version":
		version, dirty, err := migration.GetMigrationVersion(db, *migrationsPath)
		if err != nil {
			log.Fatalf("❌ Error obteniendo versión: %v", err)
		}
		if dirty {
			fmt.Printf("⚠️  Versión actual: %d (DIRTY - requiere intervención manual)\n", version)
			os.Exit(1)
		} else {
			fmt.Printf("✅ Versión actual: %d\n", version)
		}

	default:
		log.Fatalf("❌ Comando desconocido: %s. Usa: up, down, o version", *command)
	}
}
