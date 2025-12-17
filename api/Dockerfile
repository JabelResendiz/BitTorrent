# ================================
# Dockerfile para Backend API (Go)
# ================================

FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar el binario
RUN CGO_ENABLED=0 GOOS=linux go build -o api-server .

# ================================
# Imagen final
# ================================
FROM alpine:latest

WORKDIR /app

# Instalar certificados SSL
RUN apk --no-cache add ca-certificates

# Copiar el binario desde builder
COPY --from=builder /app/api-server .

# Exponer puerto 7000
EXPOSE 7000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:7000/health || exit 1

# Ejecutar la aplicación
CMD ["./api-server"]
