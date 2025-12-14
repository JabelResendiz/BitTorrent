#!/bin/bash

# Script para construir y levantar Frontend + Backend con Docker Compose

set -e

echo "=========================================="
echo "  🚀 BitTorrent - Frontend + Backend"
echo "=========================================="
echo ""

# Verificar que existe la red net
if ! docker network ls | grep -q "net"; then
    echo "📡 Creando red Docker 'net'..."
    docker network create net
fi

echo "📦 Construyendo y levantando contenedores..."
echo ""

# Construir y levantar con Docker Compose
docker compose up -d --build

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "  ✅ Contenedores iniciados"
    echo "=========================================="
    echo ""
    echo "🌐 Frontend: http://localhost:3000"
    echo "🌐 Backend:  http://localhost:7000"
    echo ""
    docker compose ps
    echo ""
    echo "📝 Ver logs: docker compose logs -f"
    echo "🛑 Detener:  docker compose down"
    echo ""
else
    echo ""
    echo "❌ Error al iniciar"
    exit 1
fi
