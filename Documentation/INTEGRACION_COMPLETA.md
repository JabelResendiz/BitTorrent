# Guía Completa de Integración - BitTorrent Web UI

## Resumen del Sistema

Este proyecto implementa una interfaz web completa para gestionar contenedores Docker que ejecutan clientes BitTorrent. El sistema consta de tres capas:

1. **Frontend**: Next.js con React (puerto 3000)
2. **Backend API**: Go con Gin framework (puerto 8090)
3. **Clientes BitTorrent**: Contenedores Docker con servidor HTTP interno (puerto 9091)

## Arquitectura Completa

```
┌──────────────────────────────────────────────────────────────────┐
│                         FRONTEND (Next.js)                        │
│                      http://localhost:3000                        │
│                                                                   │
│  - Interfaz de usuario moderna con TailwindCSS                   │
│  - Componentes React con Radix UI                                │
│  - Auto-refresh cada 3 segundos                                  │
│  - Muestra: progreso, velocidad, peers, controles                │
└────────────────────────┬─────────────────────────────────────────┘
                         │ HTTP REST API
                         ↓
┌──────────────────────────────────────────────────────────────────┐
│                      BACKEND API (Go + Gin)                       │
│                      http://localhost:8090                        │
│                                                                   │
│  Endpoints:                                                       │
│  - GET    /api/containers                                         │
│  - POST   /api/containers                                         │
│  - DELETE /api/containers/:id                                     │
│  - GET    /api/containers/:id/status                              │
│  - POST   /api/containers/:id/pause                               │
│  - POST   /api/containers/:id/resume                              │
│  - GET    /api/containers/:id/logs (WebSocket)                    │
│  - GET    /api/torrents                                           │
│  - POST   /api/torrents/upload                                    │
│  - DELETE /api/torrents/:name                                     │
│  - GET    /api/networks                                           │
└────────────────────────┬─────────────────────────────────────────┘
                         │ Docker SDK
                         ↓
┌──────────────────────────────────────────────────────────────────┐
│                       DOCKER ENGINE                               │
│                                                                   │
│  ┌────────────────────────────────────────────────────┐          │
│  │  Container 1 (client_img)                          │          │
│  │  - Cliente BitTorrent                              │          │
│  │  - HTTP Server interno (puerto 9091)               │          │
│  │    * GET  /status  → métricas en tiempo real       │          │
│  │    * POST /pause   → pausar descarga               │          │
│  │    * POST /resume  → reanudar descarga             │          │
│  └────────────────────────────────────────────────────┘          │
│                                                                   │
│  ┌────────────────────────────────────────────────────┐          │
│  │  Container 2 (client_img)                          │          │
│  │  - Otro cliente BitTorrent...                      │          │
│  └────────────────────────────────────────────────────┘          │
│                                                                   │
│  Red overlay: overlay_network                                    │
│  Volúmenes: archives/ (archivos .torrent)                        │
└──────────────────────────────────────────────────────────────────┘
```

## Estado Actual del Proyecto

### ✅ COMPLETADO

#### Backend API (`api/`)
- **api/main.go**: Servidor HTTP con Gin, CORS configurado, 15+ rutas implementadas
- **api/config.go**: Configuración centralizada (puerto 8090)
- **api/docker/client.go**: Wrapper del Docker SDK con 15 funciones
  - ListContainers, CreateContainer (con port bindings), StartContainer, StopContainer
  - RemoveContainer, GetContainer, ListImages, PullImage, CreateVolume
  - ListVolumes, RemoveVolume, ListNetworks, CreateNetwork
  - ContainerLogs, InspectImage
- **api/handlers/containers.go**: 14 handlers HTTP
  - ListContainers, CreateContainer, StopContainer, RemoveContainer
  - GetContainerStatus (proxy a container:9091/status)
  - PauseContainer (proxy a container:9091/pause)
  - ResumeContainer (proxy a container:9091/resume)
- **api/handlers/torrents.go**: Gestión de archivos .torrent
  - ListTorrents, UploadTorrent, DeleteTorrent
- **api/handlers/logs.go**: Streaming de logs en tiempo real con WebSocket
- **api/start.sh**: Script para iniciar el servidor
- **Compilación exitosa**: `go build -o api_server *.go` ✓

#### Frontend (`web/`)
- **web/components/torrent-list.tsx**: Componente principal completamente reescrito
  - Eliminados datos mock
  - Integración con API real (fetch desde http://localhost:8090/api)
  - Auto-refresh cada 3 segundos
  - Handlers para pause, resume, delete con llamadas a API
  - Muestra: progreso %, velocidad descarga/subida, peers conectados/totales, ETA
  - Botones funcionales con estados (pausa/play, eliminar)
- **web/lib/api-config.ts**: Configuración de API base URL
- **Dependencias instaladas**: Next.js 16, React 19, pnpm
- **Node.js 20.19.6** instalado (requerido para Next.js 16)

### ⏳ PENDIENTE - Implementación en Cliente BitTorrent

Para completar el sistema, necesitas implementar el servidor HTTP en el cliente BitTorrent (`src/client/`). Toda la documentación está en:

**📄 Documentation/CLIENT_HTTP_SERVER.md**

#### Archivos a crear/modificar:

1. **src/client/http_server.go** (nuevo)
   - Struct `HTTPServer` con servidor HTTP
   - Endpoints: `/status`, `/pause`, `/resume`, `/health`
   - Función `calculateStatus()` para métricas

2. **src/client/client.go** (modificar)
   - Agregar campos: `httpServer`, `paused`, `downloadSpeed`, `uploadSpeed`
   - Métodos: `Pause()`, `Resume()`, `IsPaused()`, `GetDownloadSpeed()`, etc.

3. **src/client/storage.go** (modificar)
   - Métodos: `Downloaded()`, `TotalSize()`

4. **src/client/config.go** (modificar)
   - Agregar campo `HTTPPort int`

5. **src/client/cmd/main.go** (modificar)
   - Agregar flag `--http-port`

6. **src/client/runtime/runtime_start.go** (modificar)
   - Iniciar `httpServer` en goroutine

## Flujo de Datos Completo

### Crear Nuevo Contenedor
```
Usuario → Frontend (torrent-list.tsx)
  → POST http://localhost:8090/api/containers
    Body: { name, torrentFile, networkName }
  → Backend API (handlers/containers.go: CreateContainer)
    → Docker SDK: CreateContainer con --http-port=9091 y port bindings
    → Docker SDK: StartContainer
  ← Respuesta: { id, name, status }
← Frontend muestra nuevo contenedor en lista
```

### Obtener Estado de Descarga
```
Frontend auto-refresh (cada 3s)
  → GET http://localhost:8090/api/containers/:id/status
  → Backend API (handlers/containers.go: GetContainerStatus)
    → HTTP GET http://container_ip:9091/status
    ← Container responde: {
        state, paused, progress, downloaded, total_size,
        download_speed, upload_speed, connected_peers,
        total_peers, eta
      }
  ← Backend proxy respuesta
← Frontend actualiza UI:
   - Progress bar con progress %
   - "Download: 1.2 MB/s"
   - "Peers: 5/12"
   - "ETA: 2m 15s"
```

### Pausar Descarga
```
Usuario click botón Pause → Frontend (handlePause)
  → POST http://localhost:8090/api/containers/:id/pause
  → Backend API (handlers/containers.go: PauseContainer)
    → HTTP POST http://container_ip:9091/pause
    ← Container responde: { status: "paused" }
  ← Backend proxy respuesta
← Frontend refresh automático muestra estado "Paused"
```

## Instrucciones de Uso

### 1. Iniciar Backend API

```bash
cd api
./start.sh
# O manualmente:
go run *.go
```

El servidor estará en `http://localhost:8090`

### 2. Iniciar Frontend

```bash
cd web
pnpm dev
```

La interfaz estará en `http://localhost:3000`

### 3. Implementar Cliente HTTP Server

Sigue la guía completa en `Documentation/CLIENT_HTTP_SERVER.md`:

```bash
# 1. Crear http_server.go
# 2. Modificar client.go, storage.go, config.go, etc.
# 3. Compilar cliente actualizado
cd src/client
go build -o ../../main ./cmd/main.go

# 4. Reconstruir imagen Docker
cd ../..
docker build -t client_img -f src/client/Dockerfile .

# 5. Probar localmente
./main --torrent-file=test.torrent --http-port=9091

# En otra terminal:
curl http://localhost:9091/status
curl -X POST http://localhost:9091/pause
```

### 4. Crear Contenedor desde Frontend

1. Abrir `http://localhost:3000`
2. Click en "Add Torrent"
3. Subir archivo .torrent
4. Seleccionar red overlay
5. El contenedor aparecerá en la lista con:
   - Progreso en tiempo real
   - Velocidad de descarga/subida
   - Peers conectados
   - Botones pause/resume/delete

## Endpoints API Disponibles

### Contenedores
- `GET /api/containers` - Listar todos los contenedores
- `POST /api/containers` - Crear y arrancar nuevo contenedor
  ```json
  {
    "name": "torrent_ubuntu",
    "torrentFile": "ubuntu.torrent",
    "networkName": "overlay_network"
  }
  ```
- `DELETE /api/containers/:id?force=true` - Detener y eliminar contenedor
- `GET /api/containers/:id/status` - Obtener estado en tiempo real
- `POST /api/containers/:id/pause` - Pausar descarga
- `POST /api/containers/:id/resume` - Reanudar descarga
- `GET /api/containers/:id/logs` - Stream de logs (WebSocket)

### Torrents
- `GET /api/torrents` - Listar archivos .torrent disponibles
- `POST /api/torrents/upload` - Subir nuevo .torrent (multipart/form-data)
- `DELETE /api/torrents/:name` - Eliminar archivo .torrent

### Redes
- `GET /api/networks` - Listar redes Docker

## Estructura de Datos

### TorrentItem (Frontend)
```typescript
interface TorrentItem {
  id: string                 // Container ID
  name: string               // Nombre del torrent
  containerName: string      // Nombre del contenedor
  progress: number           // 0-100
  downloadSpeed: string      // "1.2 MB/s"
  uploadSpeed: string        // "512 KB/s"
  connectedPeers: number     // 5
  totalPeers: number         // 12
  downloaded: string         // "23.5 MB"
  size: string               // "50.0 MB"
  eta: string                // "2m 15s"
  status: 'downloading' | 'seeding' | 'paused' | 'starting'
  paused: boolean
}
```

### StatusResponse (Container HTTP Server)
```go
type StatusResponse struct {
    State          string  `json:"state"`            // "downloading", "seeding", etc.
    Paused         bool    `json:"paused"`
    Progress       float64 `json:"progress"`         // 0-100
    Downloaded     int64   `json:"downloaded"`       // Bytes
    TotalSize      int64   `json:"total_size"`       // Bytes
    DownloadSpeed  int64   `json:"download_speed"`   // Bytes/s
    UploadSpeed    int64   `json:"upload_speed"`     // Bytes/s
    ConnectedPeers int     `json:"connected_peers"`
    TotalPeers     int     `json:"total_peers"`
    Eta            string  `json:"eta"`              // "2m 15s"
}
```

## Testing

### Backend API
```bash
# Test listar contenedores
curl http://localhost:8090/api/containers

# Test crear contenedor
curl -X POST http://localhost:8090/api/containers \
  -H "Content-Type: application/json" \
  -d '{"name":"test","torrentFile":"test.torrent","networkName":"overlay_network"}'

# Test obtener status
curl http://localhost:8090/api/containers/<container_id>/status

# Test pausar
curl -X POST http://localhost:8090/api/containers/<container_id>/pause
```

### Container HTTP Server (cuando esté implementado)
```bash
# Test status directo
docker exec <container_id> curl http://localhost:9091/status

# O desde host (si puerto está mapeado)
curl http://localhost:9091/status
```

## Troubleshooting

### Backend no compila
```bash
cd api
go mod tidy
go build -o api_server *.go
```

### Frontend no inicia
```bash
# Verificar Node.js
node --version  # Debe ser >= 20.9.0

# Reinstalar dependencias
cd web
rm -rf node_modules .next
pnpm install
pnpm dev
```

### CORS errors
Ya está configurado en `api/main.go` con:
- `AllowOrigins: []string{"http://localhost:3000"}`
- `AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}`

### Container no responde en puerto 9091
- Verificar que el cliente tenga implementado el HTTP server
- Verificar logs: `docker logs <container_id>`
- Verificar puerto: `docker port <container_id>`
- Verificar flag: debe iniciarse con `--http-port=9091`

## Próximos Pasos

1. **CRÍTICO**: Implementar HTTP server en cliente BitTorrent
   - Seguir `Documentation/CLIENT_HTTP_SERVER.md`
   - Crear `http_server.go`
   - Modificar `client.go`, `storage.go`, etc.
   - Compilar y probar

2. **Testing**: Probar integración completa
   - Iniciar backend API
   - Iniciar frontend
   - Crear contenedor desde UI
   - Verificar métricas en tiempo real
   - Probar pause/resume

3. **Docker**: Reconstruir imagen del cliente
   ```bash
   docker build -t client_img -f src/client/Dockerfile .
   ```

4. **Documentación adicional** (opcional):
   - Agregar screenshots del frontend
   - Crear video demo
   - Documentar troubleshooting común

## Recursos

- **Backend API**: `api/`
- **Frontend**: `web/`
- **Cliente BitTorrent**: `src/client/`
- **Documentación**:
  - `Documentation/CLIENT_HTTP_SERVER.md` - Guía implementación HTTP server
  - `Documentation/GUIA_DOS_MODOS.md` - Arquitectura general
  - `readme.md` - README principal del proyecto

## Contacto y Soporte

Para dudas sobre la implementación, revisa:
1. Los comentarios en el código fuente
2. La documentación en `Documentation/`
3. Los ejemplos de uso en este archivo
