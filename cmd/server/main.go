package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ewood-backend/config"
	"ewood-backend/internal/client/postgres"
	"ewood-backend/internal/handler"
	"ewood-backend/internal/router"
)

func main() {
	// Cargar configuración
	cfg := config.LoadConfig()

	// Conectar a PostgreSQL (si está configurado)
	if cfg.Database.PostgreSQL.Host != "" {
		if err := postgres.Connect(
			cfg.Database.PostgreSQL.Host,
			cfg.Database.PostgreSQL.Port,
			cfg.Database.PostgreSQL.User,
			cfg.Database.PostgreSQL.Password,
			cfg.Database.PostgreSQL.DBName,
			cfg.Database.PostgreSQL.SSLMode,
		); err != nil {
			log.Printf("⚠️  Advertencia: Error conectando a PostgreSQL: %v", err)
			// En producción, podrías querer hacer fatal aquí
			if cfg.Env == "production" {
				log.Fatalf("❌ Error crítico en producción: PostgreSQL no disponible: %v", err)
			}
		} else {
			defer postgres.Close()
		}
	} else {
		log.Println("ℹ️  POSTGRES_HOST no configurada - PostgreSQL no disponible")
	}

	// Inicializar WebSocket Hub
	handler.InitWebSocketHub()

	// Iniciar servidor TCP
	tcpHost := handler.GetTCPHost()
	tcpPort := handler.GetTCPPort()
	if err := handler.StartTCPServer(tcpHost, tcpPort); err != nil {
		log.Printf("⚠️  Advertencia: Error iniciando servidor TCP: %v", err)
		if cfg.Env == "production" {
			log.Fatalf("❌ Error crítico en producción: Servidor TCP no disponible: %v", err)
		}
	}

	// Configurar router
	r := router.NewRouter()

	// Manejar señales para cierre graceful
	setupGracefulShutdown()

	// Iniciar servidor HTTP
	port := cfg.Port
	fmt.Printf("🚀 Servidor HTTP escuchando en el puerto %s\n", port)
	fmt.Printf("💡 Healthcheck: http://<BACKEND_IP>:8200/check\n")
	fmt.Printf("💡 Health DB: http://<BACKEND_IP>:8200/health\n")
	fmt.Printf("💡 Variables ENV: http://<BACKEND_IP>:8200/env\n")
	fmt.Printf("💡 WebSocket: ws://<BACKEND_IP>:8200/ws/scooters\n")
	fmt.Printf("💡 API Scooters: http://<BACKEND_IP>:8200/api/scooters\n")
	if tcpHost == "" {
		fmt.Printf("💡 Servidor TCP: tcp://0.0.0.0:%s\n", tcpPort)
	} else {
		fmt.Printf("💡 Servidor TCP: tcp://%s:%s\n", tcpHost, tcpPort)
	}
	fmt.Println()

	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("❌ Error iniciando servidor: %v", err)
	}
}

// setupGracefulShutdown configura el cierre graceful
func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n🛑 Cerrando conexiones de forma segura...")
		postgres.Close()
		os.Exit(0)
	}()
}
