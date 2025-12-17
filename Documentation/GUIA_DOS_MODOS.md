# 🎯 Guía de Uso: Modo Tracker vs Modo Overlay

## Resumen de Cambios

El proyecto ahora soporta completamente **DOS modos de descubrimiento** controlados por el flag `--discovery-mode`:

1. **TRACKER** (centralizado) - Por defecto
2. **OVERLAY** (distribuido/gossip)

---

## 🏗️ Arquitectura

### Modo TRACKER (Centralizado)

```
┌─────────┐      ┌─────────┐      ┌─────────┐
│ Client1 │─────▶│ TRACKER │◀─────│ Client2 │
└─────────┘      └─────────┘      └─────────┘
     │                                  │
     └──────────────────────────────────┘
              Conexión P2P
```

- **Descubrimiento**: HTTP GET al tracker centralizado
- **Announce**: HTTP POST con eventos (started, completed, stopped)
- **Peers**: Lista devuelta por el tracker
- **Requisito**: Tracker debe estar corriendo

### Modo OVERLAY (Distribuido/Gossip)

```
┌─────────┐      ┌─────────┐      ┌─────────┐
│ Client1 │◀────▶│ Client2 │◀────▶│ Client3 │
└─────────┘      └─────────┘      └─────────┘
     ▲                                  ▲
     └──────────────────────────────────┘
         Gossip TCP JSON + P2P
```

- **Descubrimiento**: Gossip periódico entre peers
- **Announce**: Propagación epidémica (anti-entropy)
- **Peers**: Lookup en store local + consulta a bootstrap peers
- **Requisito**: Al menos 1 bootstrap peer alcanzable

---

## 🚀 Uso

### Flags Disponibles

```bash
--discovery-mode=tracker|overlay  # Modo de descubrimiento (default: tracker)
--bootstrap=host1:port1,host2:port2  # Peers bootstrap para overlay
--overlay-port=6000              # Puerto TCP para overlay (default: 6000)
--torrent=/path/to/file.torrent  # Archivo .torrent (obligatorio)
--archives=/path/to/data         # Directorio de datos
--hostname=127.0.0.1            # Hostname para announces
```

---

## 📋 Ejemplos de Uso

### Modo TRACKER (Centralizado)

#### 1. Iniciar el Tracker

```bash
cd src
go run tracker/cmd/main.go
```

Salida esperada:
```
tracker listening on :8080 interval=1800s data=tracker_data.json
```

#### 2. Iniciar Seeder (modo tracker)

```bash
go run client/cmd/main.go \
  --torrent=../archives/torrents/video.torrent \
  --archives=../archives/seeder \
  --discovery-mode=tracker \
  --hostname=127.0.0.1
```

O más simple (tracker es el default):
```bash
go run client/cmd/main.go \
  --torrent=../archives/torrents/video.torrent \
  --archives=../archives/seeder \
  --hostname=127.0.0.1
```

#### 3. Iniciar Leecher (modo tracker)

```bash
go run client/cmd/main.go \
  --torrent=../archives/torrents/video.torrent \
  --archives=../archives/leecher1 \
  --discovery-mode=tracker \
  --hostname=127.0.0.1
```

**Salida esperada en cliente:**
```
[CLIENT] === Modo de descubrimiento: TRACKER (centralizado) ===
[CLIENT] Tracker responde: map[complete:0 incomplete:2 interval:1800 peers:...]
[INFO] Announce periódico enviado (tracker)
```

---

### Modo OVERLAY (Distribuido)

#### 1. Iniciar Seeder (modo overlay)

```bash
go run client/cmd/main.go \
  --torrent=../archives/torrents/video.torrent \
  --archives=../archives/seeder \
  --discovery-mode=overlay \
  --overlay-port=6000 \
  --hostname=127.0.0.1
```

#### 2. Iniciar Leecher 1 (con bootstrap al seeder)

```bash
go run client/cmd/main.go \
  --torrent=../archives/torrents/video.torrent \
  --archives=../archives/leecher1 \
  --discovery-mode=overlay \
  --overlay-port=6001 \
  --bootstrap=127.0.0.1:6000 \
  --hostname=127.0.0.1
```

#### 3. Iniciar Leecher 2 (con bootstrap a múltiples peers)

```bash
go run client/cmd/main.go \
  --torrent=../archives/torrents/video.torrent \
  --archives=../archives/leecher2 \
  --discovery-mode=overlay \
  --overlay-port=6002 \
  --bootstrap=127.0.0.1:6000,127.0.0.1:6001 \
  --hostname=127.0.0.1
```

**Salida esperada en cliente:**
```
[CLIENT] === Modo de descubrimiento: OVERLAY/GOSSIP (distribuido) ===
[Overlay] Iniciando Overlay en :6001 con peers [127.0.0.1:6000]
Overlay gossip iniciado en :6001
[CLIENT] Overlay discovery completed; store has providers for infohash
[CLIENT] Announced to overlay, left=12345678
[INFO] Announce periódico enviado (overlay)
```

---

## 🐳 Modo TRACKER con Docker

### 1. Crear red Docker

```bash
docker network create bittorrent
```

### 2. Lanzar Tracker

```bash
docker run -d --name tracker --network bittorrent \
  -p 8080:8080 \
  tracker_img
```

### 3. Lanzar Clientes

```bash
# Seeder
docker run -d --name seeder --network bittorrent \
  -v "$(pwd)/archives/seeder":/data \
  -v "$(pwd)/archives/torrents":/torrents:ro \
  client_img \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=seeder \
  --discovery-mode=tracker

# Leecher
docker run -d --name leecher1 --network bittorrent \
  -v "$(pwd)/archives/leecher1":/data \
  -v "$(pwd)/archives/torrents":/torrents:ro \
  client_img \
  --torrent=/torrents/video.torrent \
  --archives=/data \
  --hostname=leecher1 \
  --discovery-mode=tracker
```

**NOTA**: En Docker el tracker está accesible en `http://tracker:8080/announce`

---

## 🐳 Modo OVERLAY con Docker

Usa el script existente:

```bash
./scripts/run_containers.sh
```

Este script lanza múltiples clientes en modo overlay con bootstrap encadenado.

---

## 🔍 Verificación de Logs

### Logs del Modo TRACKER

```
[CLIENT] === Modo de descubrimiento: TRACKER (centralizado) ===
[CLIENT] Tracker responde: map[complete:1 incomplete:2 interval:1800 peers:...]
[ANNOUNCE] Enviando event=started, left=12345678
[INFO] Announce periódico enviado (tracker)
[INFO] Enviando event=completed al tracker...
[INFO] Ahora soy un seeder completo
[SHUTDOWN] Enviando event=stopped al tracker...
[SHUTDOWN] Event=stopped enviado correctamente
```

### Logs del Modo OVERLAY

```
[CLIENT] === Modo de descubrimiento: OVERLAY/GOSSIP (distribuido) ===
[Overlay] Iniciando Overlay en :6000 con peers []
Overlay gossip iniciado en :6000
[CLIENT] Announced to overlay, left=12345678
Overlay providers returned: 3
[INFO] Announce periódico enviado (overlay)
[INFO] Enviando event=completed al overlay...
[INFO] Ahora soy un seeder completo (overlay)
[SHUTDOWN] Enviando event=stopped al overlay...
[SHUTDOWN] Event=stopped enviado al overlay
```

---

## 🔧 Diferencias Técnicas

| Característica | TRACKER | OVERLAY |
|----------------|---------|---------|
| **Protocolo** | HTTP/1.1 | TCP JSON |
| **Descubrimiento** | Centralizado | Distribuido (Gossip) |
| **Single Point of Failure** | ✅ Sí (tracker) | ❌ No |
| **Latencia inicial** | Baja | Media (discovery) |
| **Escalabilidad** | Media | Alta |
| **Complejidad** | Baja | Media |
| **Intervalo announces** | Configurable por tracker | Gossip periódico (8s) |
| **TTL providers** | N/A | 90 segundos |
| **Dependencias externas** | Tracker corriendo | Bootstrap peer(s) |

---

## 🐛 Troubleshooting

### Modo TRACKER

**Problema**: `Error en announce inicial: connection refused`
- **Solución**: Asegúrate de que el tracker esté corriendo en `localhost:8080`

**Problema**: `Tracker error: invalid info_hash`
- **Solución**: Verifica que el .torrent tenga la URL correcta del tracker

### Modo OVERLAY

**Problema**: `No remote providers found via overlay`
- **Solución**: Espera unos segundos para que el gossip propague la información
- Verifica que los bootstrap peers estén alcanzables

**Problema**: `Overlay discovery returned error`
- **Solución**: Verifica conectividad de red con los bootstrap peers
- Verifica que los puertos overlay no estén bloqueados

---

## 📊 Flujo de Eventos

### TRACKER
1. Cliente → Tracker: `announce?event=started`
2. Tracker → Cliente: Lista de peers
3. Cliente ↔ Peers: Conexiones P2P (handshake + peerwire)
4. Cada N segundos: Cliente → Tracker: `announce` (periódico)
5. Al completar: Cliente → Tracker: `announce?event=completed`
6. Al cerrar: Cliente → Tracker: `announce?event=stopped`

### OVERLAY
1. Cliente → Overlay: `Start()` (listener TCP)
2. Cliente → Bootstrap peers: `Discover()` (lookup remoto)
3. Cliente → Overlay local: `Announce()` (registrar provider)
4. Overlay → Bootstrap peers: Gossip periódico (cada 8s)
5. Cliente → Overlay: `Lookup()` para obtener peers
6. Cliente ↔ Peers: Conexiones P2P (handshake + peerwire)
7. Al completar/cerrar: Cliente → Overlay: `Announce()` con nuevo estado

---

## ✅ Resumen

- **Default**: Modo TRACKER (más simple, requiere tracker)
- **Para distribuido**: Usa `--discovery-mode=overlay --bootstrap=...`
- **Ambos modos** usan el mismo protocolo P2P para transferencia de piezas
- **Logging claro** indica qué modo está activo
- **Sin cambios** en el protocolo peerwire (handshake, bitfield, mensajes)

---

Generado: 8 de diciembre de 2025
