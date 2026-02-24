package migration

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations ejecuta las migraciones pendientes
func RunMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creando driver de postgres: %w", err)
	}

	// Obtener la ruta absoluta de las migraciones
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("error creando instancia de migrate: %w", err)
	}

	// Ejecutar migraciones pendientes
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("ℹ️  No hay migraciones pendientes")
			return nil
		}
		return fmt.Errorf("error ejecutando migraciones: %w", err)
	}

	log.Println("✅ Migraciones ejecutadas exitosamente")
	return nil
}

// RollbackLastMigration revierte la última migración
func RollbackLastMigration(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creando driver de postgres: %w", err)
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("error creando instancia de migrate: %w", err)
	}

	if err := m.Steps(-1); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("ℹ️  No hay migraciones para revertir")
			return nil
		}
		return fmt.Errorf("error revirtiendo migración: %w", err)
	}

	log.Println("✅ Última migración revertida exitosamente")
	return nil
}

// GetMigrationVersion obtiene la versión actual de las migraciones
func GetMigrationVersion(db *sql.DB, migrationsPath string) (uint, bool, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("error creando driver de postgres: %w", err)
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return 0, false, fmt.Errorf("error obteniendo ruta absoluta: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres",
		driver,
	)
	if err != nil {
		return 0, false, fmt.Errorf("error creando instancia de migrate: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("error obteniendo versión: %w", err)
	}

	return version, dirty, nil
}
