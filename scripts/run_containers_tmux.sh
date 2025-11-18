#!/usr/bin/env bash
set -euo pipefail

# run_containers_tmux.sh
# Levanta 1 seeder + 3 leechers en Docker y abre un tmux con 4 panes para ver logs.
# Uso: ./scripts/run_containers_tmux.sh [--no-build]

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE_NAME="client_img"
NETWORK_NAME="bittorrent"
SESSION_NAME="bittorrent"

DO_BUILD=1
if [ "${1-}" = "--no-build" ]; then
  DO_BUILD=0
fi

# Requisitos
if ! command -v docker >/dev/null 2>&1; then
  echo "Error: docker no está instalado o no está en PATH"
  exit 1
fi

if ! command -v tmux >/dev/null 2>&1; then
  echo "Warning: tmux no está instalado. El script continuará pero no creará el panel de logs. Instala tmux para ver logs en 4 panes."
  HAVE_TMUX=0
else
  HAVE_TMUX=1
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

# Build image (opcional)
# if [ $DO_BUILD -eq 1 ]; then
#   echo "Construyendo imagen docker: $IMAGE_NAME"
#   docker build -t "$IMAGE_NAME" -f "$ROOT/src/client/Dockerfile" "$ROOT"
# else
#   echo "Omitiendo build de imagen (--no-build)"
# fi

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
  --bootstrap=leecher1:6001,leecher2:6002,leecher3:6003

sleep 1

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
  --bootstrap=seeder:6000,leecher2:6002,leecher3:6003

sleep 1

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
  --bootstrap=seeder:6000,leecher1:6001,leecher3:6003

sleep 1

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
  --bootstrap=seeder:6000,leecher1:6001,leecher2:6002

# Esperar un poco para que se stabilice
sleep 1

# Preparar tmux con 4 panes para mostrar logs (si tmux está instalado)
if [ $HAVE_TMUX -eq 1 ]; then
  echo "Creando session tmux: $SESSION_NAME con 4 panes (seeder, leecher1, leecher2, leecher3)"
  # Si existe session previa, la eliminamos
  if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    tmux kill-session -t "$SESSION_NAME"
  fi
  tmux new-session -d -s "$SESSION_NAME" -n seeder
  tmux send-keys -t "$SESSION_NAME":0 'docker logs -f seeder' C-m
  tmux split-window -h -t "$SESSION_NAME":0
  tmux send-keys -t "$SESSION_NAME":0.1 'docker logs -f leecher1' C-m
  tmux split-window -v -t "$SESSION_NAME":0.0
  tmux send-keys -t "$SESSION_NAME":0.2 'docker logs -f leecher2' C-m
  tmux select-pane -t 0
  tmux split-window -v -t "$SESSION_NAME":0.1
  tmux send-keys -t "$SESSION_NAME":0.3 'docker logs -f leecher3' C-m
  tmux select-layout -t "$SESSION_NAME" tiled
  echo "Attachando a tmux session $SESSION_NAME"
  tmux attach -t "$SESSION_NAME"
else
  echo "tmux no disponible. Imprimiendo instrucciones para ver logs manualmente:"
  echo "  docker logs -f seeder"
  echo "  docker logs -f leecher1"
  echo "  docker logs -f leecher2"
  echo "  docker logs -f leecher3"
fi
