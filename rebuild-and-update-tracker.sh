#!/bin/bash
# Script para reconstruir y actualizar el tracker en Docker Swarm

set -e  # Salir si cualquier comando falla

echo "🔨 Reconstruyendo imagen del tracker..."
cd src
docker build -t tracker12 -f tracker/Dockerfile .

echo ""
echo "🔄 Actualizando servicio tracker en Docker Swarm..."
docker service update --image tracker12 tracker

echo ""
echo "⏳ Esperando a que el servicio se actualice..."
sleep 3

echo ""
echo "📊 Estado del servicio:"
docker service ps tracker --no-trunc

echo ""
echo "✅ Tracker actualizado correctamente!"
echo ""
echo "Para ver los logs en tiempo real, ejecuta:"
echo "  docker service logs tracker -f"
