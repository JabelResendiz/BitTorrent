#!/bin/bash

# Script para iniciar el API Server

cd "$(dirname "$0")"

echo "🚀 Starting BitTorrent API Server..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Verificar que Docker está corriendo
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    echo "   Please start Docker and try again"
    exit 1
fi

echo "✅ Docker is running"

# Crear directorio de torrents si no existe
mkdir -p ../archives/torrents
echo "✅ Torrents directory ready"

# Iniciar el servidor (incluir todos los archivos .go)
go run *.go
