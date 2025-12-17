#!/bin/bash

# Script para levantar el proyecto BitTorrent completo
# Uso: ./scripts/start_project.sh

set -e  # Detener si hay errores

echo "=========================================="
echo "  🚀 Iniciando Proyecto BitTorrent"
echo "=========================================="
echo ""

# Cambiar al directorio raíz del proyecto
cd "$(dirname "$0")/.."

# ============================================
# VERIFICAR HERRAMIENTAS
# ============================================
echo "🔍 Verificando herramientas necesarias..."

# Verificar Go
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go no está instalado"
    echo "   Instalar Go desde: https://go.dev/dl/"
    exit 1
fi
echo "   ✅ Go $(go version | awk '{print $3}')"

# Verificar Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Error: Node.js no está instalado"
    echo "   Instalar Node.js desde: https://nodejs.org/"
    exit 1
fi
echo "   ✅ Node.js $(node --version)"

# Verificar npm
if ! command -v npm &> /dev/null; then
    echo "❌ Error: npm no está instalado"
    echo "   npm debería venir con Node.js"
    exit 1
fi
echo "   ✅ npm $(npm --version)"

# Verificar pnpm, si no está instalarlo
if ! command -v pnpm &> /dev/null; then
    echo "   ⚠️  pnpm no está instalado, instalando..."
    npm install -g pnpm
    if [ $? -ne 0 ]; then
        echo "❌ Error al instalar pnpm"
        exit 1
    fi
fi
echo "   ✅ pnpm $(pnpm --version)"

echo ""

# ============================================
# BACKEND API
# ============================================
echo "📦 Configurando Backend API..."
cd api

# Verificar si go.mod existe
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod no encontrado en api/"
    exit 1
fi

# Descargar dependencias de Go
echo "   ⬇️  Descargando dependencias de Go..."
go mod download
go mod tidy

# Matar proceso anterior si existe
if lsof -ti:7000 > /dev/null 2>&1; then
    echo "   🔄 Deteniendo API anterior en puerto 7000..."
    lsof -ti:7000 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

# Iniciar API en background
echo "   ▶️  Iniciando API en puerto 7000..."
nohup go run *.go > /tmp/bittorrent-api.log 2>&1 &
API_PID=$!
echo "   ✅ API iniciada (PID: $API_PID)"
echo "   📝 Logs: /tmp/bittorrent-api.log"

cd ..

# ============================================
# FRONTEND
# ============================================
echo ""
echo "📦 Configurando Frontend..."
cd web

# Verificar si package.json existe
if [ ! -f "package.json" ]; then
    echo "❌ Error: package.json no encontrado en web/"
    exit 1
fi

# Instalar dependencias de Node.js
echo "   ⬇️  Descargando dependencias de Node.js..."
pnpm install

# Matar proceso anterior si existe
if lsof -ti:3000 > /dev/null 2>&1; then
    echo "   🔄 Deteniendo Frontend anterior en puerto 3000..."
    lsof -ti:3000 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

# Iniciar frontend en background
echo "   ▶️  Iniciando Frontend en puerto 3000..."
nohup pnpm dev > /tmp/bittorrent-frontend.log 2>&1 &
FRONTEND_PID=$!
echo "   ✅ Frontend iniciado (PID: $FRONTEND_PID)"
echo "   📝 Logs: /tmp/bittorrent-frontend.log"

cd ..

# ============================================
# RESUMEN
# ============================================
echo ""
echo "=========================================="
echo "  ✅ Proyecto iniciado exitosamente"
echo "=========================================="
echo ""
echo "🌐 Servicios disponibles:"
echo "   • Backend API:  http://localhost:7000"
echo "   • Frontend:     http://localhost:3000"
echo ""
echo "📋 Comandos útiles:"
echo "   • Ver logs API:      tail -f /tmp/bittorrent-api.log"
echo "   • Ver logs Frontend: tail -f /tmp/bittorrent-frontend.log"
echo "   • Detener servicios: ./scripts/stop_project.sh"
echo ""
echo "🐋 Para crear contenedores:"
echo "   • Usa la interfaz web en http://localhost:3000"
echo ""

# Esperar un momento para que los servicios inicien
sleep 3

# Verificar que los servicios estén corriendo
echo "🔍 Verificando servicios..."
if curl -s http://localhost:7000/health > /dev/null 2>&1; then
    echo "   ✅ Backend API: OK"
else
    echo "   ⚠️  Backend API: Iniciando... (puede tardar unos segundos)"
fi

if curl -s http://localhost:3000 > /dev/null 2>&1; then
    echo "   ✅ Frontend: OK"
else
    echo "   ⚠️  Frontend: Iniciando... (puede tardar unos segundos)"
fi

echo ""
echo "🎉 ¡Todo listo! Abre http://localhost:3000 en tu navegador"
