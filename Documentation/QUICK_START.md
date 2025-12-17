# 🚀 Guía de Despliegue Rápido - BitTorrent Web UI

Esta guía te llevará de 0 a tener el sistema completo funcionando en menos de 5 minutos.

---

## ⚡ Inicio Rápido (5 minutos)

### 1️⃣ Construir Imagen Docker del Cliente (1 min)

```bash
cd "/home/noel/Disco D/4to_Anno/Distribuido/BitTorrent"
docker build -t client_img -f src/client/Dockerfile .
```

**Salida esperada:**
```
Successfully built abc123def456
Successfully tagged client_img:latest
```

### 2️⃣ Iniciar Backend API (30 segundos)

```bash
# Terminal 1
cd api
./start.sh
```

**Salida esperada:**
```
Starting backend API server on port 8090...
[GIN-debug] Listening and serving HTTP on :8090
```

Verificar que funciona:
```bash
curl http://localhost:8090/api/containers
# Debe retornar: []
```

### 3️⃣ Iniciar Frontend (1 min)

```bash
# Terminal 2
cd web
pnpm dev
```

**Salida esperada:**
```
  ▲ Next.js 16.0.3
  - Local:        http://localhost:3000
  - Ready in 1.5s
```

### 4️⃣ Abrir Navegador (10 segundos)

Abrir: **http://localhost:3000**

Deberías ver la interfaz con:
- Botón "Add Torrent"
- Lista vacía de torrents
- Panel de estadísticas

---

## 📦 Crear Tu Primer Torrent

### Opción A: Usar Archivo de Prueba

```bash
cd "/home/noel/Disco D/4to_Anno/Distribuido/BitTorrent"

# Crear archivo de prueba
echo "Hola Mundo desde BitTorrent!" > archives/seeder/test.txt

# Crear .torrent
mktorrent -a http://tracker:8080/announce \
  -o archives/torrents/test.torrent \
  archives/seeder/test.txt
```

### Opción B: Usar Archivo Existente

```bash
# Si ya tienes un archivo (video, ISO, etc.)
cp /path/to/tu/archivo.mp4 archives/seeder/

# Crear .torrent
mktorrent -a http://tracker:8080/announce \
  -o archives/torrents/archivo.torrent \
  archives/seeder/archivo.mp4
```

---

## 🎮 Usar la Interfaz Web

### Paso 1: Subir Torrent

1. Click en **"Add Torrent"**
2. Seleccionar archivo `.torrent` de `archives/torrents/`
3. Seleccionar red: `overlay_network`
4. Click **"Create Container"**

### Paso 2: Ver Progreso

Después de crear el contenedor, verás:

```
┌─────────────────────────────────────────────┐
│ test.txt                                    │
│ container_test_123                          │
│ Status: Downloading                         │
│                                             │
│ ▓▓▓▓▓▓▓▓▓░░░░░░░░░░░░░░░ 35%              │
│                                             │
│ 1.2 MB / 3.5 MB                            │
│                                             │
│ ⬇ Download: 512 KB/s                       │
│ ⬆ Upload:   128 KB/s                       │
│ 👥 Peers:    3/5                            │
│ ⏱ ETA:      4s                              │
│                                             │
│ [❚❚ Pause]  [🗑 Delete]                     │
└─────────────────────────────────────────────┘
```

La página se actualiza automáticamente cada 3 segundos.

### Paso 3: Controlar Descarga

- **Pausar**: Click en botón ❚❚ Pause
  - La descarga se detiene
  - Estado cambia a "Paused"
  - Velocidad = 0

- **Reanudar**: Click en botón ▶ Play
  - La descarga continúa
  - Estado cambia a "Downloading"
  - Velocidad se recupera

- **Eliminar**: Click en botón 🗑 Delete
  - Confirmar en el diálogo
  - El contenedor se detiene y elimina

---

## 🧪 Verificar que Todo Funciona

### Test 1: Backend API

```bash
# Listar contenedores
curl http://localhost:8090/api/containers

# Listar torrents
curl http://localhost:8090/api/torrents

# Crear red overlay (si no existe)
curl -X POST http://localhost:8090/api/networks \
  -H "Content-Type: application/json" \
  -d '{"name":"overlay_network","driver":"overlay"}'
```

### Test 2: Servidor HTTP del Cliente

Después de crear un contenedor desde la UI:

```bash
# Obtener ID del contenedor
CONTAINER_ID=$(docker ps --filter "name=torrent_" --format "{{.ID}}" | head -n 1)

# Consultar puerto mapeado
PORT=$(docker port $CONTAINER_ID 9091 | cut -d: -f2)

# Probar servidor HTTP
curl http://localhost:$PORT/status | jq

# Pausar desde línea de comandos
curl -X POST http://localhost:$PORT/pause

# Reanudar
curl -X POST http://localhost:$PORT/resume
```

### Test 3: Auto-refresh del Frontend

1. Abrir `http://localhost:3000`
2. Crear un contenedor con un torrent
3. Observar cómo las métricas se actualizan automáticamente:
   - Progreso incrementa
   - Velocidad cambia
   - ETA disminuye

No necesitas refrescar la página manualmente.

---

## 🔍 Monitoreo Avanzado

### Ver Logs del Contenedor en Tiempo Real

Desde la línea de comandos:

```bash
# Logs del backend API
cd api
tail -f nohup.out

# Logs de un contenedor específico
docker logs -f <container_id>

# Logs del frontend (si lo iniciaste con nohup)
cd web
tail -f nohup.out
```

### Monitorear Todos los Contenedores

```bash
# Listar todos los contenedores activos
docker ps --filter "ancestor=client_img"

# Ver recursos usados
docker stats $(docker ps -q --filter "ancestor=client_img")
```

### Inspeccionar Estado Detallado

```bash
# Información completa del contenedor
docker inspect <container_id> | jq

# Solo ver comandos y configuración
docker inspect <container_id> | jq '.[0].Config.Cmd'

# Ver redes conectadas
docker inspect <container_id> | jq '.[0].NetworkSettings.Networks'
```

---

## 🛑 Detener Todo

### Opción 1: Detener Servicios Individuales

```bash
# Detener frontend (Ctrl+C en terminal o)
pkill -f "next dev"

# Detener backend
pkill -f "go run"
# O si usaste ./start.sh:
kill $(cat api/api.pid)

# Detener contenedores BitTorrent
docker stop $(docker ps -q --filter "ancestor=client_img")
```

### Opción 2: Script de Limpieza Completa

```bash
cd "/home/noel/Disco D/4to_Anno/Distribuido/BitTorrent"
./scripts/stop.sh

# O manualmente:
docker stop $(docker ps -q --filter "ancestor=client_img")
docker rm $(docker ps -aq --filter "ancestor=client_img")
```

---

## 🔧 Solución de Problemas Comunes

### Problema: "Frontend no se conecta al backend"

**Solución:**
```bash
# 1. Verificar que backend está corriendo
curl http://localhost:8090/api/containers

# 2. Si no responde, revisar logs
cd api
cat nohup.out

# 3. Reiniciar backend
pkill -f "go run"
./start.sh
```

### Problema: "Puerto 8090 ya está en uso"

**Solución:**
```bash
# Ver qué proceso usa el puerto
sudo lsof -i :8090

# Matar proceso
kill -9 <PID>

# O cambiar puerto en api/config.go
# Cambiar APIPort = ":8090" a ":8091"
```

### Problema: "Error al crear contenedor"

**Solución:**
```bash
# 1. Verificar que la imagen existe
docker images | grep client_img

# 2. Si no existe, reconstruir
docker build -t client_img -f src/client/Dockerfile .

# 3. Verificar que la red overlay existe
docker network ls | grep overlay_network

# 4. Si no existe, crear
docker network create -d overlay overlay_network
```

### Problema: "Contenedor creado pero no responde en puerto 9091"

**Solución:**
```bash
# 1. Ver logs del contenedor
docker logs <container_id>

# 2. Verificar que el servidor HTTP inició
docker logs <container_id> | grep "Iniciando servidor HTTP"

# 3. Probar desde dentro del contenedor
docker exec <container_id> wget -q -O - http://localhost:9091/health

# 4. Verificar port mapping
docker port <container_id>
```

---

## 📈 Siguiente Nivel

### Crear Múltiples Clientes

```bash
# Desde la UI, crear 3 contenedores:
# 1. torrent_client_1
# 2. torrent_client_2  
# 3. torrent_client_3

# Todos compartirán piezas entre sí automáticamente
# Ver en tiempo real cómo se distribuyen los bloques
```

### Modo Overlay (Distribuido)

Al crear el contenedor desde la UI, el sistema automáticamente:
- Usa overlay network para descubrimiento P2P
- Implementa protocolo Gossip para difundir información
- Distribuye bloques con Round-Robin entre peers

No necesitas configurar nada adicional.

### Integrar con Docker Swarm

Para un cluster multi-nodo, ver:
- [DOCKER_SWARM_GUIDE.md](../Documentation/DOCKER_SWARM_GUIDE.md)

---

## 📚 Recursos Adicionales

### Documentación
- **Guía Completa**: [INTEGRACION_COMPLETA.md](../Documentation/INTEGRACION_COMPLETA.md)
- **Implementación HTTP**: [IMPLEMENTACION_HTTP_SERVER.md](../Documentation/IMPLEMENTACION_HTTP_SERVER.md)
- **Arquitectura**: [ARQUITECTURA_P2P.md](../Documentation/ARQUITECTURA_P2P.md)

### Scripts Útiles
- `scripts/test_http_server.sh` - Test del servidor HTTP
- `scripts/run.sh` - Modo clásico (sin UI)
- `scripts/stop.sh` - Detener todo

### Endpoints API
- **Documentación Postman**: Próximamente
- **OpenAPI/Swagger**: Próximamente

---

## ✅ Checklist de Verificación

Antes de reportar un problema, verifica:

- [ ] Docker está instalado y corriendo
- [ ] Go 1.22+ está instalado
- [ ] Node.js 20+ está instalado
- [ ] pnpm está instalado
- [ ] Imagen `client_img` existe (`docker images | grep client_img`)
- [ ] Backend API responde (`curl http://localhost:8090/api/containers`)
- [ ] Frontend carga (`http://localhost:3000`)
- [ ] Puerto 8090 está libre
- [ ] Puerto 3000 está libre
- [ ] Red overlay existe (`docker network ls | grep overlay_network`)

---

## 🎉 ¡Listo!

Ahora tienes un sistema BitTorrent distribuido completamente funcional con interfaz web moderna.

**Disfruta descargando y compartiendo archivos de forma descentralizada!** 🚀

---

**¿Necesitas ayuda?**
- 📖 Revisa la [documentación completa](../Documentation/)
- 🐛 Abre un [issue en GitHub](https://github.com/JabelResendiz/BitTorrent/issues)
- 💬 Consulta el código fuente para entender la implementación
