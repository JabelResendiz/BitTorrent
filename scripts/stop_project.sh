#!/bin/bash

# Script para detener todos los servicios del proyecto BitTorrent
# Uso: ./scripts/stop_project.sh

echo "=========================================="
echo "  🛑 Deteniendo Proyecto BitTorrent"
echo "=========================================="
echo ""

# Detener Backend API (puerto 7000)
if lsof -ti:7000 > /dev/null 2>&1; then
    echo "🔴 Deteniendo Backend API (puerto 7000)..."
    lsof -ti:7000 | xargs kill -9 2>/dev/null
    echo "   ✅ Backend detenido"
else
    echo "   ℹ️  Backend no está corriendo"
fi

# Detener Frontend (puerto 3000)
if lsof -ti:3000 > /dev/null 2>&1; then
    echo "🔴 Deteniendo Frontend (puerto 3000)..."
    lsof -ti:3000 | xargs kill -9 2>/dev/null
    echo "   ✅ Frontend detenido"
else
    echo "   ℹ️  Frontend no está corriendo"
fi

echo ""
echo "=========================================="
echo "  ✅ Servicios detenidos"
echo "=========================================="
echo ""
echo "💡 Para reiniciar: ./scripts/start_project.sh"
