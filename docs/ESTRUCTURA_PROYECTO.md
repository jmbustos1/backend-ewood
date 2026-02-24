# 🏗️ Estructura del Proyecto

Este proyecto sigue una arquitectura limpia (Clean Architecture) para mantener el código organizado y fácil de mantener.

## 📁 Estructura de Directorios

```
backend-ewood/
├── cmd/
│   └── server/
│       └── main.go              # Punto de entrada de la aplicación
├── internal/                    # Código interno del proyecto
│   ├── handler/                 # Handlers HTTP
│   │   ├── check.go            # Handler para /check
│   │   ├── health.go           # Handler para /health
│   │   └── env.go              # Handler para /env
│   ├── router/                  # Configuración de rutas
│   │   └── router.go           # Definición de rutas HTTP
│   ├── service/                 # Lógica de negocio (futuro)
│   └── client/                  # Clientes de bases de datos y servicios externos
│       ├── postgres/           # Cliente de PostgreSQL
│       │   └── postgres.go
│       └── mongodb/            # Cliente de MongoDB (futuro)
├── config/                      # Configuración de la aplicación
│   └── config.go               # Carga de variables de entorno
├── docs/                        # Documentación del proyecto
├── terraform/                   # Infraestructura como código
├── docker-compose.yml           # Docker Compose para producción
├── docker-compose-local.yml     # Docker Compose para desarrollo local
├── Dockerfile                   # Imagen Docker
├── Makefile                     # Comandos útiles
├── deploy.sh                    # Script de despliegue
├── go.mod                       # Dependencias Go
└── README.md                    # Documentación principal
```

## 🎯 Principios de la Arquitectura

### 1. **Separación de Responsabilidades**

- **cmd/**: Punto de entrada de la aplicación. Solo contiene `main.go` que inicializa y ejecuta el servidor.
- **internal/handler/**: Maneja las peticiones HTTP. Los handlers son delgados y delegan la lógica de negocio a los servicios.
- **internal/service/**: Contiene la lógica de negocio. Los servicios son independientes de HTTP y pueden ser reutilizados.
- **internal/client/**: Clientes para servicios externos (bases de datos, APIs, etc.). Abstraen la comunicación con servicios externos.
- **internal/router/**: Configuración de rutas. Centraliza todas las definiciones de rutas HTTP.
- **config/**: Configuración de la aplicación. Carga y valida variables de entorno.

### 2. **Capas de la Arquitectura**

```
┌─────────────────────────────────────┐
│         HTTP Layer (handlers)       │
│  - Recibe requests HTTP             │
│  - Valida input                     │
│  - Llama a servicios                │
│  - Retorna respuestas HTTP          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Business Logic (services)      │
│  - Lógica de negocio                │
│  - Reglas de negocio                │
│  - Validaciones complejas           │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Data Access (clients)          │
│  - Conexiones a bases de datos      │
│  - Queries                          │
│  - Clientes de APIs externas        │
└─────────────────────────────────────┘
```

### 3. **Independencia de Frameworks**

El código de negocio no depende de frameworks HTTP específicos. Esto permite:
- Cambiar el framework HTTP sin afectar la lógica de negocio
- Probar la lógica de negocio sin necesidad de un servidor HTTP
- Reutilizar la lógica en diferentes contextos (CLI, jobs, etc.)

## 📝 Convenciones de Nomenclatura

- **Handlers**: Nombres descriptivos seguidos de `Handler` (ej: `CheckHandler`, `HealthHandler`)
- **Services**: Nombres descriptivos seguidos de `Service` (ej: `UserService`, `BikeService`)
- **Clients**: Nombres descriptivos seguidos del tipo (ej: `postgres`, `mongodb`)
- **Config**: Tipos descriptivos seguidos de `Config` (ej: `PostgreSQLConfig`, `MongoDBConfig`)

## 🔄 Flujo de una Petición HTTP

1. **HTTP Request** → `internal/router/router.go`
2. **Router** → Enruta a `internal/handler/`
3. **Handler** → Llama a `internal/service/` (si existe lógica de negocio)
4. **Service** → Usa `internal/client/` para acceder a datos
5. **Client** → Ejecuta queries o llamadas a APIs
6. **Response** → Se retorna por las capas hasta llegar al cliente HTTP

## 🚀 Agregar Nuevas Funcionalidades

### Agregar un nuevo endpoint:

1. Crear handler en `internal/handler/nuevo_handler.go`:
```go
package handler

func NuevoHandler(w http.ResponseWriter, r *http.Request) {
    // Lógica del handler
}
```

2. Agregar ruta en `internal/router/router.go`:
```go
mux.HandleFunc("/nueva-ruta", handler.NuevoHandler)
```

### Agregar lógica de negocio:

1. Crear service en `internal/service/nuevo_service.go`:
```go
package service

type NuevoService struct {
    // Dependencias
}

func (s *NuevoService) MetodoDeNegocio() error {
    // Lógica de negocio
}
```

2. Usar el service en el handler:
```go
func NuevoHandler(w http.ResponseWriter, r *http.Request) {
    service := service.NewNuevoService()
    result := service.MetodoDeNegocio()
    // Responder
}
```

## 📚 Referencias

Esta estructura está inspirada en:
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- Estructuras de proyectos Go de referencia (forest/backend, forest-bikes)

