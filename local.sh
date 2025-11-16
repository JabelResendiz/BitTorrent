#!/bin/bash

set -e  

# VARIABLES GLOBALES
TRACKER_IMAGE="tracker_img:latest"
TRACKER_CONTAINER="tracker"
CLIENT_IMAGE="client_img:latest"
TRACKER_PORT="8081:8080"
NETWORK_NAME="net"
BASE_VOLUME_PATH="${BT_VOLUME_PATH:?Debe definir BT_VOLUME_PATH}"
SEED_TORRENT="${BT_SEED_TORRENT:?Debe definir BT_SEED_TORRENT}"
SEED_MOVIE="${BT_SEED_MOVIE:?Debe definir BT_SEED_MOVIE}"
CLIENT_COUNT=${1:-3}

#####
if ! docker network ls | grep -q "$NETWORK_NAME"; then
  echo -e "\e[32mCreando red '$NETWORK_NAME'...\e[0m"
  docker network create "$NETWORK_NAME" 
fi

check_image() {
  local image="$1"
  if docker image inspect "$image" > /dev/null 2>&1; then
    echo -e "\e[33m✔ La imagen '$image' ya existe. No se reconstruira\e[0m"
    return 0
  else
    echo -e "\e[32m La imagen '$image' no existe.\e[0m"
    return 1
  fi
}


check_container() {
  local container="$1"
  if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
    echo -e "\e[33m✔ El contenedor '$container' ya existe. Se eliminara.\e[0m"
    docker rm -f "$container" > /dev/null 2>&1
  else
    echo -e "\e[32m⚠ El contenedor '$container' no existe.\e[0m"
  
  return 0
  fi
}

echo -e "\e[32m🔨 Construyendo imagen del tracker...\e[0m"
cd src

if ! check_image "$TRACKER_IMAGE"; then
    docker build -t "$TRACKER_IMAGE" -f tracker/Dockerfile .
fi

echo -e "\e[32m🔨 Construyendo imagen del client...\e[0m"

if ! check_image "$CLIENT_IMAGE"; then
    docker build -t "$CLIENT_IMAGE" -f client/Dockerfile .
fi

echo -e "\e[32m📡 Desplegando tracker...\e[0m"

if check_container "$TRACKER_CONTAINER"; then
    docker run  -d --name "$TRACKER_CONTAINER"   --network "$NETWORK_NAME"   --publish "$TRACKER_PORT" "$TRACKER_IMAGE"
fi


echo -e "\e[34m⏳ Esperando que el tracker se inicie...\e[0m"
sleep 15

echo -e "\e[32mDesplegando clientes ...\e[0m"

for ((i=1;i<$CLIENT_COUNT+1;i++)); do
    CLIENT_NAME="client$i"
    VOLUME_PATH="${BASE_VOLUME_PATH}${i}"
    HOSTNAME="$CLIENT_NAME"


    echo -e "\e[34mIniciando $CLIENT_NAME ...\e[0m"
    echo -e "\e[34m - Volumen: $VOLUME_PATH\e[0m"
    echo -e "\e[34m - Hostname: $HOSTNAME\e[0m"


    mkdir -p "$VOLUME_PATH"

    echo -e "\e[33m📄 Copiando torrent desde el seeder...\e[0m"
    cp "$SEED_TORRENT" "$VOLUME_PATH/video2.torrent"

    if [[ $i -eq 1 ]]; then
        echo -e "\e[33m🌱 Este es el seeder → copiando archivo completo...\e[0m"
        cp "$SEED_MOVIE" "$VOLUME_PATH/Lecture 2 - Divide and Conquer.mp4"
    fi



    if check_container "$CLIENT_NAME"; then
        docker run -d \
        --name "$CLIENT_NAME" \
        --network "$NETWORK_NAME" \
        -v "$VOLUME_PATH":/app/src/archives \
        "$CLIENT_IMAGE" \
        --torrent="/app/src/archives/video2.torrent" \
        --archives="/app/src/archives" \
        --hostname="$HOSTNAME"
    fi
    

    echo -e "\e[32m✅ $CLIENT_NAME desplegado correctamente\e[0m"
    echo "---"

done


echo ""
echo -e "\e[32m✅ Tracker actualizado correctamente!\e[0m"
echo ""
echo -e "\e[32mLOGS EN TIEMPO REAL DEL TRACKER\e[0m"
docker logs -f "$TRACKER_CONTAINER"
