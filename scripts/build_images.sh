#!/bin/bash

# Script para compilar las imágenes Docker del proyecto BitTorrent
# Uso: ./scripts/build_images.sh

set -e  # Detener si hay errores

echo "=========================================="
echo "  Compilando Imágenes Docker"
echo "=========================================="
echo ""

# Cambiar al directorio raíz del proyecto
cd "$(dirname "$0")/.."

# Compilar imagen del tracker
echo "📦 Compilando imagen del tracker..."
docker build -t tracker_img -f src/tracker/Dockerfile .
echo "✅ Imagen tracker_img compilada"
echo ""

# Compilar imagen del cliente
echo "📦 Compilando imagen del cliente..."
docker build -t client_img -f src/client/Dockerfile .
echo "✅ Imagen client_img compilada"
echo ""

echo "=========================================="
echo "  ✅ Imágenes compiladas exitosamente"
echo "=========================================="
echo ""
echo "Imágenes disponibles:"
docker images | grep -E "tracker_img|client_img"
