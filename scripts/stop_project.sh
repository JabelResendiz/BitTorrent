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
echo "🔴 Deteniendo Frontend..."
if lsof -ti:3000 > /dev/null 2>&1; then
    echo "   Deteniendo proceso en puerto 3000..."
    lsof -ti:3000 | xargs kill -9 2>/dev/null
fi

# Matar procesos de Next.js y pnpm relacionados
if pgrep -f "next dev" > /dev/null 2>&1; then
    echo "   Deteniendo procesos de Next.js..."
    pkill -9 -f "next dev" 2>/dev/null
fi

if pgrep -f "pnpm.*dev" > /dev/null 2>&1; then
    echo "   Deteniendo procesos de pnpm dev..."
    pkill -9 -f "pnpm.*dev" 2>/dev/null
fi

# Verificar que el puerto 3000 esté libre
sleep 1
if lsof -ti:3000 > /dev/null 2>&1; then
    echo "   ⚠️  Puerto 3000 aún ocupado, forzando..."
    lsof -ti:3000 | xargs kill -9 2>/dev/null
    sleep 1
fi

if ! lsof -ti:3000 > /dev/null 2>&1; then
    echo "   ✅ Frontend detenido"
else
    echo "   ❌ No se pudo liberar el puerto 3000"
fi

echo ""
echo "=========================================="
echo "  ✅ Servicios detenidos"
echo "=========================================="
echo ""
echo "💡 Para reiniciar: ./scripts/start_project.sh"
