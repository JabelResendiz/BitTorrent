# Guía: Ejecutar Cliente BitTorrent en Docker Swarm con DNS

## 🔍 Problema Resuelto

**Antes:** Los clientes reportaban IPs internas de contenedor (`10.0.1.8`) que no eran accesibles entre nodos del swarm.

**Ahora:** Los clientes usan **nombres de contenedor (hostnames)** que Docker Swarm resuelve automáticamente vía DNS interno. **No necesitas port mapping ni IPs del host.**

## 📋 Prerequisitos

1. Tener Docker Swarm configurado
2. Red overlay attachable creada: `net`
3. Tracker corriendo como servicio
4. Imagen del cliente construida: `client12`

## 🚀 Cómo Usar

### Método Simple (Recomendado)

```bash
# En MÁQUINA A
docker run -it --rm \
  --name client1 \
  --network net \
  -v ~/Desktop/volumen:/app/src/archives \
  client12 \
  --torrent="/app/src/archives/vid.torrent" \
  --archives="/app/src/archives" \
  --hostname="client1"

# En MÁQUINA B  
docker run -it --rm \
  --name client2 \
  --network net \
  -v ~/Desktop/volumen:/app/src/archives \
  client12 \
  --torrent="/app/src/archives/vid.torrent" \
  --archives="/app/src/archives" \
  --hostname="client2"
```

**Nota:** Docker Swarm resolverá automáticamente `client1` y `client2` a las IPs correctas dentro de la red overlay.

## ✅ Verificación

Después de ejecutar los clientes, deberías ver:

```
[ANNOUNCE] Enviando event=started, left=4016308224
Tracker responde: map[complete:1 incomplete:1 interval:1800 peers:[...]]

Peer: client2:36891
Conectado al peer, handshake OK  ← ✅ Conexión exitosa usando hostname!
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
# event=started from client1 (ih=... pid=... left=...)
# event=started from client2 (ih=... pid=... left=...)
```

## 🐛 Troubleshooting

### Problema: "no such host" o "dial tcp: lookup client2"
**Solución:** Asegúrate de que ambos contenedores están en la misma red overlay `net`

### Problema: "connection refused"
**Solución:** Verifica que ambos clientes estén ejecutándose y hayan enviado announce al tracker

### Problema: Tracker no responde
**Solución:** Verifica que el servicio tracker está corriendo:
```bash
docker service ls
docker service ps tracker
```

## 📝 Notas Importantes

1. **Sin Port Mapping:** No necesitas `-p` porque Docker Swarm maneja las conexiones internas

2. **Hostname Obligatorio:** Debes pasar `--hostname` con el mismo valor que `--name` del contenedor

3. **Red Overlay:** Todos los contenedores deben estar en la misma red overlay attachable

4. **DNS Interno:** Docker Swarm resuelve automáticamente los nombres de contenedor a IPs

5. **Formato Non-Compact:** El tracker detecta automáticamente si hay hostnames y usa formato non-compact

## 🎯 Ventajas de Esta Solución

✅ **Más simple:** No necesitas conocer las IPs de los hosts  
✅ **Más robusto:** Si un contenedor se reinicia, el hostname sigue siendo el mismo  
✅ **Cloud-native:** Aprovecha el DNS interno de Docker Swarm  
✅ **Sin port mapping:** Las conexiones son directas entre contenedores  
✅ **Compatible:** Sigue funcionando con IPs numéricas para clientes externos
