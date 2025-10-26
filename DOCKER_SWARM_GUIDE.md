# Guía: Ejecutar Cliente BitTorrent en Docker Swarm

## 🔍 Problema Resuelto

**Antes:** Los clientes reportaban IPs internas de contenedor (`10.0.1.8`) que no eran accesibles entre nodos del swarm.

**Ahora:** Los clientes reportan la IP del host usando el flag `--external-ip`, permitiendo conexiones peer-to-peer entre diferentes máquinas.

## 📋 Prerequisitos

1. Tener Docker Swarm configurado
2. Red overlay attachable creada: `net`
3. Tracker corriendo como servicio
4. Imagen del cliente construida: `client12`

## 🚀 Cómo Usar

### Opción 1: Usando el Script (Recomendado)

```bash
# Hacer el script ejecutable
chmod +x docker-run-client.sh

# Ejecutar en MÁQUINA A (ejemplo: IP 192.168.1.10)
./docker-run-client.sh 192.168.1.10 43629 client1

# Ejecutar en MÁQUINA B (ejemplo: IP 192.168.1.20)
./docker-run-client.sh 192.168.1.20 37443 client2
```

### Opción 2: Comando Manual

#### En MÁQUINA A (ebur02):

```bash
# 1. Obtener IP del host
HOST_IP=$(hostname -I | awk '{print $1}')
echo "Mi IP: $HOST_IP"

# 2. Ejecutar cliente
docker run -it --rm \
  --name client1 \
  --network net \
  -p 43629:43629 \
  -v ~/Desktop/volumen:/app/src/archives \
  client12 \
  --torrent="/app/src/archives/vid.torrent" \
  --archives="/app/src/archives" \
  --external-ip="$HOST_IP" \
  --port=43629
```

#### En MÁQUINA B (TANIA):

```bash
# 1. Obtener IP del host
HOST_IP=$(hostname -I | awk '{print $1}')
echo "Mi IP: $HOST_IP"

# 2. Ejecutar cliente
docker run -it --rm \
  --name client2 \
  --network net \
  -p 37443:37443 \
  -v ~/Desktop/volumen:/app/src/archives \
  client12 \
  --torrent="/app/src/archives/vid.torrent" \
  --archives="/app/src/archives" \
  --external-ip="$HOST_IP" \
  --port=37443
```

## ✅ Verificación

Después de ejecutar los clientes, deberías ver:

```
[ANNOUNCE] Usando IP externa: 192.168.1.10
[ANNOUNCE] Enviando event=started, left=4016308224
Tracker responde: map[complete:1 incomplete:1 interval:1800 peers:...]

Peer: 192.168.1.20:37443
Conectado al peer, handshake OK  ← ✅ Conexión exitosa!
```

## 🔧 Reconstruir Imagen del Cliente

Si hiciste cambios al código, reconstruye la imagen:

```bash
cd src
docker build -t client12 -f client/Dockerfile .
```

## 📊 Logs del Tracker

Para ver si los clientes se registran correctamente:

```bash
# Ver logs del servicio tracker
docker service logs tracker -f

# Deberías ver:
# event=started from 192.168.1.10 (ih=... pid=... left=...)
# event=started from 192.168.1.20 (ih=... pid=... left=...)
```

## 🐛 Troubleshooting

### Problema: "connection refused"
**Solución:** Asegúrate de usar `-p <PORT>:<PORT>` y `--external-ip`

### Problema: "port already in use"
**Solución:** Usa un puerto diferente o detén el contenedor anterior

### Problema: "no route to host"
**Solución:** Verifica que ambas máquinas están en la misma red y pueden hacer ping

```bash
# Desde MÁQUINA A
ping 192.168.1.20

# Desde MÁQUINA B
ping 192.168.1.10
```

## 📝 Notas Importantes

1. **Port Mapping:** El puerto interno del contenedor debe coincidir con el puerto mapeado (`-p PORT:PORT`)

2. **IP Externa:** Usa la IP de la interfaz de red que conecta ambas máquinas (no localhost ni 127.0.0.1)

3. **Firewall:** Asegúrate de que los puertos estén abiertos en el firewall:
   ```bash
   sudo ufw allow <PORT>/tcp
   ```

4. **Obtener IP automáticamente:**
   ```bash
   # Linux
   hostname -I | awk '{print $1}'
   
   # O con ip
   ip route get 1 | awk '{print $7;exit}'
   ```

## 🎯 Alternativa: Network Host Mode

Si tienes problemas, puedes usar `--network host` (más simple pero menos aislado):

```bash
docker run -it --rm \
  --network host \
  -v ~/Desktop/volumen:/app/src/archives \
  client12 \
  --torrent="/app/src/archives/vid.torrent" \
  --archives="/app/src/archives"
```

**Nota:** En este modo NO necesitas `--external-ip` ni port mapping.
