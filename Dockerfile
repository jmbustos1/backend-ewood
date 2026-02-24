# Imagen base con Go
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copiar el código fuente
COPY go.mod go.sum ./
RUN go mod download

# Copiar todo el código
COPY . .

# Compilar los binarios de la aplicación
RUN go build -o server ./cmd/server
RUN go build -o migrate ./cmd/migrate

# Imagen final ligera
FROM alpine:latest

# Instalar ca-certificates para conexiones HTTPS
RUN apk --no-cache add ca-certificates

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar los binarios desde la imagen anterior
COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY --from=builder /app/migrations ./migrations

# Exponer los puertos HTTP y TCP
EXPOSE 8080 8201

# Ejecutar el servidor TCP
CMD ["./server"]
