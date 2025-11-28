#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE_NAME="client_img"
NETWORK_NAME="bittorrent"
SESSION_NAME="bittorrent"

DO_BUILD=1
if [ "${1-}" = "--no-build" ]; then
  DO_BUILD=0
fi

# Crear network si no existe
if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "Creando red docker: $NETWORK_NAME"
  docker network create "$NETWORK_NAME"
else
  echo "Red $NETWORK_NAME ya existe"
fi

# Crear directorios de datos
mkdir -p "$ROOT/archives/seeder" "$ROOT/archives/leecher1" "$ROOT/archives/leecher2" "$ROOT/archives/leecher3" "$ROOT/archives/torrents"

# Verificar .torrent
if [ ! -f "$ROOT/archives/torrents/video.torrent" ]; then
  echo "Error: no encontré $ROOT/archives/torrents/video.torrent"
  echo "Copia tu .torrent a ese path y vuelve a ejecutar."
  exit 1
fi


# Borrar contenedores previos si existen
for c in seeder leecher1 leecher2 leecher3; do
  if docker ps -a --format '{{.Names}}' | grep -q "^${c}$"; then
    echo "Eliminando contenedor previo: $c"
    docker rm -f "$c" >/dev/null 2>&1 || true
  fi
done

# Lanzar contenedores (detached)
echo "Lanzando seeder..."
docker run -d --name seeder --network "$NETWORK_NAME" \
  -v "$ROOT/archives/seeder":/data \
  -v "$ROOT/archives/torrents":/torrents:ro \
  -p 6000:6000 \
  "$IMAGE_NAME" \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=seeder \
  --discovery-mode=overlay \
  --overlay-port=6000 \

sleep 3

echo "Lanzando leecher1..."
docker run -d --name leecher1 --network "$NETWORK_NAME" \
  -v "$ROOT/archives/leecher1":/data \
  -v "$ROOT/archives/torrents":/torrents:ro \
  -p 6001:6001 \
  "$IMAGE_NAME" \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=leecher1 \
  --discovery-mode=overlay \
  --overlay-port=6001 \
  --bootstrap=seeder:6000

sleep 3

echo "Lanzando leecher2..."
docker run -d --name leecher2 --network "$NETWORK_NAME" \
  -v "$ROOT/archives/leecher2":/data \
  -v "$ROOT/archives/torrents":/torrents:ro \
  -p 6002:6002 \
  "$IMAGE_NAME" \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=leecher2 \
  --discovery-mode=overlay \
  --overlay-port=6002 \
  --bootstrap=leecher1:6001

sleep 3

echo "Lanzando leecher3..."
docker run -d --name leecher3 --network "$NETWORK_NAME" \
  -v "$ROOT/archives/leecher3":/data \
  -v "$ROOT/archives/torrents":/torrents:ro \
  -p 6003:6003 \
  "$IMAGE_NAME" \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=leecher3 \
  --discovery-mode=overlay \
  --overlay-port=6003 \
  --bootstrap=leecher2:6002

sleep 3

echo "Lanzando leecher4..."
docker run -d --name leecher4 --network "$NETWORK_NAME" \
  -v "$ROOT/archives/leecher4":/data \
  -v "$ROOT/archives/torrents":/torrents:ro \
  -p 6004:6004 \
  "$IMAGE_NAME" \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=leecher4 \
  --discovery-mode=overlay \
  --overlay-port=6004 \
  --bootstrap=leecher3:6003

# # Esperar un poco para que se stabilice
# sleep 1

