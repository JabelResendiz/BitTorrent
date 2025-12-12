# ============================================
# COMPILAR IMÁGENES (hacer primero)
# ============================================

# Compilar imagen del tracker
docker build -t tracker_img -f src/tracker/Dockerfile .

# Compilar imagen del cliente
docker build -t client_img -f src/client/Dockerfile .

# Crear red compartida
docker network create net

# ============================================
# TRACKERS DISTRIBUIDOS (3 trackers sincronizados)
# ============================================

# Tracker 1
docker run -d \
  --name tracker1 \
  --hostname tracker1 \
  --network net \
  --publish 8081:8080 \
  --publish 9091:9090 \
  tracker_img \
  -sync-peers "tracker2:9090,tracker3:9090"

# Tracker 2
docker run -d \
  --name tracker2 \
  --hostname tracker2 \
  --network net \
  --publish 8082:8080 \
  --publish 9092:9090 \
  tracker_img \
  -sync-peers "tracker1:9090,tracker3:9090"

# Tracker 3
docker run -d \
  --name tracker3 \
  --hostname tracker3 \
  --network net \
  --publish 8083:8080 \
  --publish 9093:9090 \
  tracker_img \
  -sync-peers "tracker1:9090,tracker2:9090"

# Ver logs de sincronización
docker logs -f tracker1

# ============================================
# MODO STANDALONE (un solo tracker, sin sincronización)
# ============================================
docker run -d \
  --name tracker \
  --network net \
  --publish 8081:8080 \
  tracker_img

# ============================================
# CLIENTES
# ============================================

# Cliente en modo tracker (puede conectarse a cualquier tracker)
# Opción 1: Conectar a tracker1
docker run -it --rm \
  --name client1 \
  --network net \
  -v ~/Desktop/peers/1:/app/src/archives \
  client_img \
  --torrent="/app/src/archives/ST.torrent" \
  --archives="/app/src/archives" \
  --hostname="client1" \
  --discovery-mode=tracker

# Opción 2: Conectar a tracker2
docker run -it --rm \
  --name client2 \
  --network net \
  -v ~/Desktop/peers/2:/app/src/archives \
  client_img \
  --torrent="/app/src/archives/ST.torrent" \
  --archives="/app/src/archives" \
  --hostname="client2" \
  --discovery-mode=tracker

# Opción 3: Conectar a tracker3
docker run -it --rm \
  --name client3 \
  --network net \
  -v ~/Desktop/peers/3:/app/src/archives \
  client_img \
  --torrent="/app/src/archives/ST.torrent" \
  --archives="/app/src/archives" \
  --hostname="client3" \
  --discovery-mode=tracker

# NOTA: Los clientes se conectan usando el DNS de Docker
# - tracker1:8080/announce
# - tracker2:8080/announce  
# - tracker3:8080/announce
# Los 3 trackers se sincronizan automáticamente cada 15 segundos
#
# CAMBIOS IMPORTANTES:
# - Ya NO necesitas pasar -node-id (se usa el hostname del contenedor automáticamente)
# - Ya NO necesitas pasar -data (se crea automáticamente como /data/<hostname>_data.json)
# - En -sync-peers usas los nombres de contenedor directamente (tracker1, tracker2, tracker3)
# ============================================
# Cliente en modo overlay (necesita overlay-port y bootstrap)
# ============================================
docker run -it --rm \
  --name client_overlay1 \
  --network net \
  -v ~/Desktop/peers/overlay1:/app/src/archives \
  -p 6001:6001 \
  client_img \
  --torrent="/app/src/archives/ST.torrent" \
  --archives="/app/src/archives" \
  --hostname="client_overlay1" \
  --discovery-mode=overlay \
  --overlay-port=6001 \
  --bootstrap=client1:6000

# ============================================
# RESUMEN DE CONEXIÓN
# ============================================
# 
# MODO TRACKER DISTRIBUIDO:
# - Los clientes configuran en el .torrent: announce=http://tracker1:8080/announce
# - Pueden usar tracker1, tracker2 o tracker3 (todos se sincronizan)
# - DNS interno de Docker: tracker1:8080, tracker2:8080, tracker3:8080
# - Puertos externos: localhost:8081, localhost:8082, localhost:8083
# - Node-ID: Se obtiene automáticamente del hostname del contenedor (tracker1, tracker2, tracker3)
# - Archivo de datos: Se crea automáticamente en /data/<hostname>_data.json
#
# MODO STANDALONE:
# - Un solo tracker: tracker:8080
# - DNS interno: tracker:8080
# - Puerto externo: localhost:8081
# - Node-ID automático: nombre del contenedor
#
# SINCRONIZACIÓN (entre trackers):
# - Puerto 9090 para comunicación entre trackers
# - Gossip push cada 15 segundos
# - Los trackers se encuentran usando DNS: tracker1:9090, tracker2:9090, tracker3:9090
# - En los logs verás el node-id de cada tracker (tracker1, tracker2, tracker3)