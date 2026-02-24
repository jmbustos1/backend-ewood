package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"
)

var (
	DB  *sql.DB
	GDB *gorm.DB // GORM database instance
)

// Connect establece la conexión a PostgreSQL
// Es idempotente: funciona tanto en local como en producción
func Connect(host, port, user, password, dbname, sslmode string) error {
	// Construir connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	// Abrir conexión
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("error abriendo conexión a PostgreSQL: %w", err)
	}

	// Configurar pool de conexiones
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Verificar la conexión
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("error haciendo ping a PostgreSQL: %w", err)
	}

	DB = db

	// Inicializar GORM
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("error inicializando GORM: %w", err)
	}

	GDB = gormDB
	fmt.Println("✅ Conectado a PostgreSQL exitosamente")
	fmt.Printf("   Host: %s:%s\n", host, port)
	fmt.Printf("   Database: %s\n", dbname)
	return nil
}

// Close cierra la conexión a PostgreSQL
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// Ping verifica que la conexión esté activa
func Ping() error {
	if DB == nil {
		return fmt.Errorf("PostgreSQL no está inicializado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return DB.PingContext(ctx)
}

