#!/bin/bash

set -e  

# VARIABLES GLOBALES
TRACKER_IMAGE="tracker_img:latest"
TRACKER_CONTAINER="tracker"
CLIENT_IMAGE="client_img:latest"
CLIENT_CONTAINER="client"

TRACKER_PORT="8081:8080"

VOLUME_PATH="$HOME/Desktop/volumen"
HOSTNAME="client2"

NETWORK_NAME="net"

#####

if ! docker network ls | grep -q "$NETWORK_NAME"; then
  echo "Creando red overlay '$NETWORK_NAME'..."
  docker network create -d overlay "$NETWORK_NAME" --attachable
fi


echo "🔨 Construyendo imagen del tracker..."
cd src
docker build -t "$TRACKER_IMAGE" -f tracker/Dockerfile .

echo "🔨 Construyendo imagen del client..."
cd src
docker build -t "$CLIENT_IMAGE" -f client/Dockerfile .

echo "📡 Desplegando tracker..."
docker service create  --name "$TRACKER_CONTAINER"   --network "$NETWORK_NAME"   --publish "$TRACKER_PORT" "$TRACKER_IMAGE"

docker run -it --rm \
  --name "$CLIENT_CONTAINER" \
  --network "$NETWORK_NAME" \
  -v "$VOLUME_PATH":/app/src/archives \
  "$CLIENT_IMAGE" \
  --torrent="/app/src/archives/vid.torrent" \
  --archives="/app/src/archives" \
  --hostname="$HOSTNAME"


echo ""
echo "✅ Tracker actualizado correctamente!"
echo ""
echo "LOGS EN TIEMPO REAL"
docker service logs "$TRACKER_CONTAINER"
