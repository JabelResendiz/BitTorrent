# 🔥 BitTorrent con Interfaz Web

Sistema BitTorrent distribuido con descubrimiento P2P mediante **Docker**, **Go** y **overlay networks**, ahora con una **interfaz web moderna** para gestión y monitoreo en tiempo real.

![Estado](https://img.shields.io/badge/Estado-Producción-success)
![Docker](https://img.shields.io/badge/Docker-Enabled-blue)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8)
![Next.js](https://img.shields.io/badge/Next.js-16-black)

---

## 🌟 Características Principales

### Sistema BitTorrent Core
- ✅ Cliente BitTorrent completo implementado en Go
- ✅ Soporte para múltiples trackers con failover automático
- ✅ Descubrimiento P2P mediante overlay networks (Gossip protocol)
- ✅ Distribución de bloques con Round-Robin entre peers
- ✅ Verificación SHA-1 de integridad por pieza
- ✅ Reanudación de descargas interrumpidas
- ✅ Modo tracker centralizado + modo overlay distribuido

### Interfaz Web y API (NUEVO)
- ✅ **Interfaz web moderna** con Next.js y React
- ✅ **Backend API REST** en Go con Docker SDK
- ✅ **Monitoreo en tiempo real** de descargas
- ✅ **Control de pause/resume** desde la UI
- ✅ **Métricas detalladas**: progreso %, velocidad, peers, ETA
- ✅ **Auto-refresh** cada 3 segundos
- ✅ **Gestión de contenedores** Docker desde la web
- ✅ **Streaming de logs** en tiempo real (WebSocket)

---

## 🏗️ Arquitectura del Sistema

```
┌──────────────────────────────────────────────────────────────┐
│                    FRONTEND (Next.js)                         │
│                  http://localhost:3000                        │
│                                                               │
│  - Interfaz moderna con TailwindCSS                          │
│  - Gestión de torrents y contenedores                        │
│  - Métricas en tiempo real                                   │
└────────────────────────┬─────────────────────────────────────┘
                         │ REST API
                         ↓
┌──────────────────────────────────────────────────────────────┐
│                   BACKEND API (Go + Gin)                      │
│                  http://localhost:8090                        │
│                                                               │
│  - Gestión de contenedores Docker                            │
│  - Proxy a servidores HTTP de clientes                       │
│  - Gestión de archivos .torrent                              │
│  - Streaming de logs                                          │
└────────────────────────┬─────────────────────────────────────┘
                         │ Docker SDK
                         ↓
┌──────────────────────────────────────────────────────────────┐
│                      DOCKER ENGINE                            │
│                                                               │
│  ┌────────────────────────────────────────────┐              │
│  │  Container BitTorrent Client               │              │
│  │  - Cliente BitTorrent (Go)                 │              │
│  │  - HTTP Server interno (puerto 9091)       │              │
│  │    * GET /status → métricas               │              │
│  │    * POST /pause → pausar descarga        │              │
│  │    * POST /resume → reanudar              │              │
│  └────────────────────────────────────────────┘              │
│                                                               │
│  Redes: overlay_network (gossip P2P)                         │
│  Volúmenes: archives/ (archivos .torrent y descargas)        │
└──────────────────────────────────────────────────────────────┘
```

---

## 🚀 Inicio Rápido

### Opción 1: Sistema Completo con Interfaz Web

```bash
# 1. Clonar el proyecto
git clone https://github.com/JabelResendiz/BitTorrent.git
cd BitTorrent

# 2. Construir imagen del cliente
docker build -t client_img -f src/client/Dockerfile .

# 3. Iniciar Backend API
cd api
./start.sh

# 4. En otra terminal, iniciar Frontend
cd web
pnpm install
pnpm dev

# 5. Abrir navegador
# Frontend: http://localhost:3000
# Backend API: http://localhost:8090
```

### Opción 2: Modo Clásico (Sin UI)

```bash
# 1. Clonar y construir
git clone https://github.com/JabelResendiz/BitTorrent.git
cd BitTorrent/src
docker build -t client_img -f client/Dockerfile .

# 2. Crear archivo .torrent
mktorrent -a http://tracker:8080/announce \
  -o archives/torrents/video.torrent \
  archives/seeder/video.mp4

# 3. Ejecutar con scripts
cd ..
./scripts/run.sh
```

---

## 📱 Interfaz Web

### Capturas de Pantalla (Conceptual)

#### Dashboard Principal
- Lista de torrents activos
- Progreso visual con barras
- Métricas: velocidad descarga/subida, peers conectados, ETA
- Botones: Pause, Resume, Delete

#### Características de la UI
- **Auto-refresh**: Se actualiza cada 3 segundos automáticamente
- **Responsive**: Funciona en desktop y móvil
- **Dark Mode Ready**: Diseño moderno con soporte para tema oscuro
- **Componentes Reutilizables**: Construida con Radix UI

### Endpoints del Backend API

#### Contenedores
```bash
# Listar contenedores
GET /api/containers

# Crear contenedor
POST /api/containers
Body: {"name": "torrent_1", "torrentFile": "video.torrent", "networkName": "overlay_network"}

# Obtener estado
GET /api/containers/:id/status

# Pausar/Reanudar
POST /api/containers/:id/pause
POST /api/containers/:id/resume

# Eliminar
DELETE /api/containers/:id?force=true

# Logs en tiempo real (WebSocket)
GET /api/containers/:id/logs
```

#### Torrents
```bash
# Listar archivos .torrent
GET /api/torrents

# Subir .torrent
POST /api/torrents/upload
Content-Type: multipart/form-data

# Eliminar
DELETE /api/torrents/:name
```

---

## 🛠️ Instalación Detallada

### Prerrequisitos

- **Docker** >= 20.10
- **Go** >= 1.22 (para compilar desde source)
- **Node.js** >= 20.9.0 (para frontend)
- **pnpm** (gestor de paquetes)

### Instalación de Dependencias

```bash
# Backend API
cd api
go mod download

# Frontend
cd ../web
pnpm install
```

### Configuración

#### Backend API (`api/config.go`)
```go
const (
    APIPort = ":8090"
)
```

#### Frontend (`web/lib/api-config.ts`)
```typescript
export const API_BASE_URL = 'http://localhost:8090/api'
```

---

## 📖 Documentación

### Documentación Principal
- **[INTEGRACION_COMPLETA.md](Documentation/INTEGRACION_COMPLETA.md)** - Guía completa del sistema
- **[CLIENT_HTTP_SERVER.md](Documentation/CLIENT_HTTP_SERVER.md)** - Implementación del servidor HTTP
- **[IMPLEMENTACION_HTTP_SERVER.md](Documentation/IMPLEMENTACION_HTTP_SERVER.md)** - Estado actual y testing

### Documentación Técnica
- **[ARQUITECTURA_P2P.md](Documentation/ARQUITECTURA_P2P.md)** - Arquitectura del sistema P2P
- **[OVERLAY_GOSSIP_IMPLEMENTATION.md](Documentation/OVERLAY_GOSSIP_IMPLEMENTATION.md)** - Protocolo Gossip
- **[ROUND_ROBIN_IMPLEMENTATION.md](Documentation/ROUND_ROBIN_IMPLEMENTATION.md)** - Distribución de bloques
- **[DOCKER_SWARM_GUIDE.md](Documentation/DOCKER_SWARM_GUIDE.md)** - Despliegue en cluster

### Guías de Uso
- **[GUIA_DOS_MODOS.md](Documentation/GUIA_DOS_MODOS.md)** - Modo tracker vs overlay
- **[comandos.md](Documentation/comandos.md)** - Comandos útiles
- **[LOGGING_ROUND_ROBIN.md](Documentation/LOGGING_ROUND_ROBIN.md)** - Sistema de logging

---

## 🧪 Testing

### Test del Servidor HTTP del Cliente

```bash
# Compilar cliente
cd src/client
go build -o ../../client_binary ./cmd/main.go

# Ejecutar cliente
cd ../..
./client_binary \
  --torrent=archives/test.torrent \
  --http-port=9091 \
  --hostname=$(hostname -I | awk '{print $1}')

# En otra terminal, ejecutar script de prueba
./scripts/test_http_server.sh
```

Salida esperada:
```
================================================
   Prueba del Servidor HTTP - Cliente BitTorrent
================================================

[1] Verificando servidor HTTP...
✓ Servidor HTTP está corriendo

[2] Consultando estado...
Estado actual:
{
  "state": "downloading",
  "paused": false,
  "progress": 45.32,
  "downloaded": 23456789,
  "total_size": 51773349,
  "download_speed": 1048576,
  "upload_speed": 524288,
  "connected_peers": 5,
  "total_peers": 5,
  "eta": "25s"
}

[3] Pausando descarga...
✓ Descarga pausada correctamente

[4] Reanudando descarga...
✓ Descarga reanudada correctamente

[5] Monitoreando progreso (5 segundos)...
  [1/5] Progreso: 45.32% | Velocidad: 1.0 MiB/s | Peers: 5 | ETA: 25s
  [2/5] Progreso: 46.10% | Velocidad: 1.1 MiB/s | Peers: 5 | ETA: 23s
  ...

================================================
✓ Todas las pruebas completadas exitosamente
================================================
```

### Test del Backend API

```bash
# Terminal 1: Iniciar backend
cd api
./start.sh

# Terminal 2: Ejecutar pruebas
# Listar contenedores
curl http://localhost:8090/api/containers

# Crear contenedor
curl -X POST http://localhost:8090/api/containers \
  -H "Content-Type: application/json" \
  -d '{"name":"test","torrentFile":"test.torrent","networkName":"overlay_network"}'

# Obtener estado
curl http://localhost:8090/api/containers/<container_id>/status | jq
```

### Test del Frontend

```bash
# Terminal 1: Backend
cd api && ./start.sh

# Terminal 2: Frontend
cd web && pnpm dev

# Navegador: http://localhost:3000
# 1. Subir archivo .torrent
# 2. Ver contenedor creándose
# 3. Observar progreso en tiempo real
# 4. Probar pause/resume
```

---

## 🔧 Uso Avanzado

### Flags del Cliente BitTorrent

```bash
./client_binary \
  --torrent=<path>              # Archivo .torrent (obligatorio)
  --archives=./archives         # Directorio de archivos
  --hostname=<ip>               # IP para announces (NAT/Docker)
  --discovery-mode=overlay      # tracker|overlay
  --bootstrap=<host:port>       # Bootstrap peers para overlay
  --overlay-port=6000           # Puerto overlay network
  --http-port=9091              # Puerto servidor HTTP (NUEVO)
```

### Ejemplo: Cliente con HTTP Server

```bash
docker run -d \
  --name torrent_client_1 \
  --network overlay_network \
  -v "$(pwd)/archives:/app/archives" \
  -p 9091:9091 \
  client_img \
  --torrent=/app/archives/video.torrent \
  --archives=/app/archives \
  --hostname=torrent_client_1 \
  --discovery-mode=overlay \
  --bootstrap=client2:6000,client3:6000 \
  --http-port=9091
```

### Consultar Estado del Contenedor

```bash
# Desde el host
curl http://localhost:9091/status

# Desde otro contenedor
docker exec torrent_client_1 wget -q -O - http://localhost:9091/status
```

---

## 🐛 Troubleshooting

### Frontend no se conecta al backend
```bash
# Verificar que el backend esté corriendo
curl http://localhost:8090/api/containers

# Verificar configuración CORS en api/main.go
# Debe incluir: AllowOrigins: []string{"http://localhost:3000"}
```

### Puerto 8090 ocupado
```bash
# Cambiar puerto en api/config.go
const APIPort = ":8091"

# Actualizar frontend en web/lib/api-config.ts
export const API_BASE_URL = 'http://localhost:8091/api'
```

### Contenedor no responde en puerto 9091
```bash
# Verificar logs del contenedor
docker logs <container_id>

# Verificar puerto mapeado
docker port <container_id>

# Verificar flag HTTP port
docker inspect <container_id> | jq '.[0].Config.Cmd'
```

### Errores de compilación en Go
```bash
cd api
go mod tidy
go build -o api_server *.go

cd ../src/client
go mod tidy
go build -o ../../client_binary ./cmd/main.go
```

---

## 📊 Métricas y Monitoreo

### Métricas Disponibles (Endpoint `/status`)

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `state` | string | Estado: `"starting"`, `"downloading"`, `"seeding"`, `"completed"`, `"paused"` |
| `paused` | boolean | Si está pausado |
| `progress` | float64 | Porcentaje 0-100 |
| `downloaded` | int64 | Bytes descargados |
| `total_size` | int64 | Tamaño total |
| `download_speed` | int64 | Bytes/segundo |
| `upload_speed` | int64 | Bytes/segundo |
| `connected_peers` | int | Peers conectados |
| `total_peers` | int | Peers conocidos |
| `eta` | string | Tiempo estimado |

### Ejemplo de Monitoreo Continuo

```bash
# Con watch y jq
watch -n 1 'curl -s http://localhost:9091/status | jq'

# Script personalizado
while true; do
  curl -s http://localhost:9091/status | \
    jq -r '"Progress: \(.progress)% | Speed: \(.download_speed) B/s | Peers: \(.connected_peers)"'
  sleep 1
done
```

---

## 🤝 Contribuir

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

---

## 📝 Licencia

Este proyecto está bajo la Licencia MIT. Ver archivo [LICENSE](LICENSE) para más detalles.

---

## 👥 Autores

- **JabelResendiz** - [GitHub](https://github.com/JabelResendiz)

---

## 🙏 Agradecimientos

- Protocolo BitTorrent (BEP 0003)
- Docker y Go community
- Next.js y React ecosystem
- Contributors y testers

---

## 📞 Soporte

- 📧 Email: [tu-email@ejemplo.com]
- 🐛 Issues: [GitHub Issues](https://github.com/JabelResendiz/BitTorrent/issues)
- 📖 Docs: [Documentation/](Documentation/)

---

**¡Disfruta de tu sistema BitTorrent distribuido con interfaz web!** 🚀
