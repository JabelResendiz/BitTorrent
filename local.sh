#!/bin/bash

set -e  

# VARIABLES GLOBALES
TRACKER_IMAGE="tracker_img:latest"
TRACKER_CONTAINER="tracker"
CLIENT_IMAGE="client_img:latest"
CLIENT_CONTAINER1="client1"
CLIENT_CONTAINER2="client2"

TRACKER_PORT="8081:8080"

VOLUME_PATH1="$HOME/jabel/volumen1"
VOLUME_PATH2="$HOME/jabel/volumen2"
HOSTNAME2="client2"
HOSTNAME2="client1"
NETWORK_NAME="net"

#####

if ! docker network ls | grep -q "$NETWORK_NAME"; then
  echo "Creando red '$NETWORK_NAME'..."
  docker network create "$NETWORK_NAME" 
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
  --name "$CLIENT_CONTAINER1" \
  --network "$NETWORK_NAME" \
  -v "$VOLUME_PATH":/app/src/archives \
  "$CLIENT_IMAGE" \
  --torrent="/app/src/archives/video.torrent" \
  --archives="/app/src/archives" \
  --hostname="$HOSTNAME1"

docker run -it --rm \
  --name "$CLIENT_CONTAINER2" \
  --network "$NETWORK_NAME" \
  -v "$VOLUME_PATH":/app/src/archives \
  "$CLIENT_IMAGE" \
  --torrent="/app/src/archives/video.torrent" \
  --archives="/app/src/archives" \
  --hostname="$HOSTNAME2"

echo ""
echo "✅ Tracker actualizado correctamente!"
echo ""
echo "LOGS EN TIEMPO REAL"
docker service logs "$TRACKER_CONTAINER"
