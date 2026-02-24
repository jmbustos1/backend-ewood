.PHONY: dockerup dockerdown build-no-cache dropdb-and-container test-tcp migrate-up migrate-down migrate-version migrate-up-host migrate-down-host run-simulate-dev-tcp help

# Ejecutar contenedor localmente
dockerup:
	docker compose -f docker-compose-local.yml up

# Detener contenedor local
dockerdown:
	docker compose -f docker-compose-local.yml down

# Reconstruir sin cache
build-no-cache:
	docker compose -f docker-compose-local.yml build --no-cache

# Detener y eliminar volúmenes
dropdb-and-container:
	docker compose -f docker-compose-local.yml down -v

# Variables de configuración TCP (pueden ser sobrescritas)
TCP_HOST ?= localhost
TCP_PORT ?= 8201

# Probar conexión TCP (requiere que el servidor esté corriendo)
test-tcp:
	@echo "Probando conexión TCP al servidor..."
	nc -v $(TCP_HOST) $(TCP_PORT) || echo "Asegúrate de que el servidor esté corriendo en $(TCP_HOST):$(TCP_PORT)"

# Ejecutar el servidor localmente
run-local:
	go run ./cmd/server

# Ejecutar tests
test:
	go test ./...

# Verificar variables de entorno en el contenedor local
check-env:
	@echo "🔍 Verificando variables de entorno en el contenedor local..."
	@curl -s http://localhost:8200/env | jq || echo "Error: ¿Está corriendo el servidor? Ejecuta 'make dockerup'"

# Ver variables de entorno sin formato (para debug)
check-env-raw:
	@curl -s http://localhost:8200/env

# Ver variables de entorno directamente del contenedor
check-env-container:
	@echo "🔍 Variables de entorno del contenedor (sin contraseñas):"
	@docker exec ewood-challenge env | grep -E "POSTGRES|MONGODB|ENV|PORT" | grep -v PASSWORD | sort || echo "Error: ¿Está corriendo el contenedor?"

# ================================
# Comandos de Migración
# ================================

# Ejecutar migraciones pendientes (en contenedor local)
# El contenedor tiene acceso a la red de Docker y puede conectarse a postgres:5432
migrate-up:
	@echo "🔄 Ejecutando migraciones en contenedor local..."
	@docker exec ewood-challenge /app/migrate -command=up -path=/app/migrations || echo "❌ Error: ¿Está corriendo el contenedor? Ejecuta 'make dockerup'"

# Revertir última migración (en contenedor local)
migrate-down:
	@echo "🔄 Revirtiendo última migración en contenedor local..."
	@docker exec ewood-challenge /app/migrate -command=down -path=/app/migrations || echo "❌ Error: ¿Está corriendo el contenedor? Ejecuta 'make dockerup'"

# Ver versión actual de migraciones (en contenedor local)
migrate-version:
	@echo "📋 Versión actual de migraciones en contenedor:"
	@docker exec ewood-challenge /app/migrate -command=version -path=/app/migrations || echo "❌ Error: ¿Está corriendo el contenedor? Ejecuta 'make dockerup'"

# Ejecutar migraciones desde el host (requiere que postgres esté expuesto en localhost:5432)
# Útil si quieres ejecutar sin contenedor, pero necesitas configurar POSTGRES_HOST=localhost
migrate-up-host:
	@echo "🔄 Ejecutando migraciones desde el host (conectando a localhost:5432)..."
	@POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=ewood_user POSTGRES_PASSWORD=ewood_pass POSTGRES_DB=ewood_db POSTGRES_SSLMODE=disable go run ./cmd/migrate -command=up -path=./migrations

# Revertir migración desde el host
migrate-down-host:
	@echo "🔄 Revirtiendo migración desde el host..."
	@POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=ewood_user POSTGRES_PASSWORD=ewood_pass POSTGRES_DB=ewood_db POSTGRES_SSLMODE=disable go run ./cmd/migrate -command=down -path=./migrations

# ================================
# Comandos de Simulación
# ================================

# Ejecutar simulación de scooter por TCP con hot reload
# Uso: make run-simulate-dev-tcp ID=1
# Personalizar: make run-simulate-dev-tcp ID=1 TCP_HOST=3.237.198.242 TCP_PORT=8201
run-simulate-dev-tcp:
	@if [ -z "$(ID)" ]; then \
		echo "❌ Error: Se requiere el ID del scooter"; \
		echo "   Uso: make run-simulate-dev-tcp ID=1"; \
		exit 1; \
	fi
	@echo "🛴 Iniciando simulación TCP para scooter ID: $(ID)"
	@echo "🌐 Conectando a $(TCP_HOST):$(TCP_PORT)"
	@TCP_HOST=$(TCP_HOST) TCP_PORT=$(TCP_PORT) python3 scripts/simulate_scooter_tcp.py $(ID)

# Mostrar ayuda
help:
	@echo "📋 Comandos disponibles:"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make dockerup              - Iniciar contenedores"
	@echo "  make dockerdown            - Detener contenedores"
	@echo "  make build-no-cache        - Reconstruir sin cache"
	@echo "  make dropdb-and-container  - Eliminar volúmenes y contenedores"
	@echo ""
	@echo "🚀 Servidor:"
	@echo "  make run-local             - Ejecutar servidor localmente"
	@echo "  make test                  - Ejecutar tests"
	@echo ""
	@echo "🔌 TCP:"
	@echo "  make test-tcp              - Probar conexión TCP"
	@echo "  make run-simulate-dev-tcp ID=X  - Simular scooter por TCP"
	@echo "    Ejemplo: make run-simulate-dev-tcp ID=1 TCP_HOST=3.237.198.242 TCP_PORT=8201"
	@echo ""
	@echo "🔄 Migraciones:"
	@echo "  make migrate-up            - Ejecutar migraciones"
	@echo "  make migrate-down          - Revertir última migración"
	@echo "  make migrate-version       - Ver versión actual"
	@echo ""
	@echo "🔍 Verificación:"
	@echo "  make check-env             - Verificar variables de entorno"
	@echo "  make check-env-container  - Ver variables del contenedor"
