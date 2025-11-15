#!/bin/bash
set -e

TRACKER_CONTAINER="tracker"
CLIENT_BASE="client"

echo -e "\e[31m🛑 Deteniendo y eliminando el tracker...\e[0m"
if docker ps -a --format '{{.Names}}' | grep -q "^${TRACKER_CONTAINER}$"; then
    docker rm -f "$TRACKER_CONTAINER"
    echo -e "\e[31m✔ Tracker eliminado\e[0m"
else
    echo -e "\e[33m⚠ No existe el tracker\e[0m"
fi

echo -e "\e[31m🛑 Deteniendo y eliminando clientes...\e[0m"
for container in $(docker ps -a --format '{{.Names}}' | grep "^${CLIENT_BASE}[0-9]\+$"); do
    docker rm -f "$container"
    echo -e "\e[31m✔ $container eliminado\e[0m"
done

echo -e "\e[32m✅ Todos los contenedores detenidos y eliminados\e[0m"
