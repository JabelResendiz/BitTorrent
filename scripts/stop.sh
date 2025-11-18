#!/bin/bash
set -e

TRACKER_CONTAINER="tracker"
CLIENT_BASE="client"
BASE_VOLUME_PATH="${BT_VOLUME_PATH:-./volumen}"

echo -e "\e[32m📁 Usando BASE_VOLUME_PATH = $BASE_VOLUME_PATH\e[0m"

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


echo -e "\e[31m🗑 Eliminando carpetas de volúmenes...\e[0m"

shopt -s nullglob

for folder in ${BASE_VOLUME_PATH}[0-9]*; do
    if [[ -d "$folder" && "$folder" =~ ${BASE_VOLUME_PATH}[0-9]+$ ]]; then
        rm -rf "$folder"
        echo -e "\e[31m✔ Carpeta '$folder' eliminada\e[0m"
    fi
done
shopt -u nullglob