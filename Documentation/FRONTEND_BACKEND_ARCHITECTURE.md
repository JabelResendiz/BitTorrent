# 🏗️ Arquitectura Frontend + Backend + Docker

Documentación técnica completa del sistema de interfaz web para BitTorrent

---

## 📊 **Diagrama de arquitectura completa**

```
┌─────────────────────────────────────────────────────────────────┐
│                        USUARIO                                   │
│                     (Navegador Web)                              │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     │ HTTP (REST API)
                     │ WebSocket (logs)
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                    FRONTEND (Next.js)                            │
│                   Puerto: 3000                                   │
├─────────────────────────────────────────────────────────────────┤
│  ├── app/page.tsx                  (Página principal)            │
│  ├── components/                                                 │
│  │   ├── add-torrent-form.tsx     (Formulario crear container)  │
│  │   ├── torrent-list.tsx         (Lista de contenedores)       │
│  │   └── stats-overview.tsx       (Estadísticas)                │
│  └── services/api.ts               (Cliente HTTP)                │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     │ fetch() / WebSocket
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                    BACKEND API (Go)                              │
│                   Puerto: 8090                                   │
├─────────────────────────────────────────────────────────────────┤
│  ├── main.go                       (Servidor HTTP + CORS)        │
│  ├── handlers/                                                   │
│  │   ├── containers.go             (CRUD contenedores)           │
│  │   ├── torrents.go               (Gestión .torrent)            │
│  │   └── logs.go                   (WebSocket streaming)         │
│  └── docker/                                                     │
│      └── client.go                 (Docker SDK wrapper)          │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     │ Docker Socket
                     │ /var/run/docker.sock
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                    DOCKER ENGINE                                 │
├─────────────────────────────────────────────────────────────────┤
│  ├── Networks:                                                   │
│  │   └── bittorrent (overlay)                                   │
│  │                                                               │
│  ├── Containers:                                                 │
│  │   ├── seeder      (client_img)  [Nodo con archivo completo]  │
│  │   ├── leecher1    (client_img)  [Nodo descargando]           │
│  │   ├── leecher2    (client_img)  [Nodo descargando]           │
│  │   └── leecher3    (client_img)  [Nodo descargando]           │
│  │                                                               │
│  └── Volumes:                                                    │
│      ├── archives/seeder/    → /data (en seeder)                │
│      ├── archives/leecher1/  → /data (en leecher1)              │
│      ├── archives/leecher2/  → /data (en leecher2)              │
│      └── archives/torrents/  → /torrents:ro (read-only)         │
└─────────────────────────────────────────────────────────────────┘
                     │
                     │ Overlay Network (Gossip P2P)
                     ▼
         ┌───────────────────────────┐
         │  P2P BitTorrent Protocol  │
         │  (Peerwire + Gossip)      │
         └───────────────────────────┘
```

---

## 🔄 **Flujo de datos completo**

### **Ejemplo: Usuario crea un leecher desde el navegador**

```
1. USUARIO
   │ Hace clic en "Submit" en el formulario
   │ Datos: { containerName: "leecher1", port: "6001", ... }
   ▼

2. FRONTEND (React Component)
   │ Función: handleSubmit()
   │ fetch('http://localhost:8090/api/containers', {
   │   method: 'POST',
   │   body: JSON.stringify(config)
   │ })
   ▼

3. BACKEND API (main.go)
   │ Gin Router recibe: POST /api/containers
   │ CORS middleware: ✅ Origin permitido
   │ Enruta a: handlers.CreateContainer()
   ▼

4. HANDLER (containers.go)
   │ Función: CreateContainer(c *gin.Context)
   │ - Valida el JSON recibido
   │ - Construye comando: ["--torrent=/torrents/video.torrent", ...]
   │ - Construye binds: ["/home/user/archives/leecher1:/data"]
   │ - Llama a: dockerClient.CreateContainer(config)
   ▼

5. DOCKER CLIENT (docker/client.go)
   │ Función: CreateContainer(config)
   │ - ctx := context.Background()
   │ - cli.ContainerCreate(ctx, containerConfig, hostConfig, ...)
   │ - cli.ContainerStart(ctx, containerID, ...)
   │ - return containerID
   ▼

6. DOCKER ENGINE
   │ - Crea el contenedor con imagen client_img
   │ - Monta volúmenes
   │ - Conecta a red "bittorrent"
   │ - Ejecuta: /app/main --torrent=... --hostname=leecher1 ...
   │ - Contenedor inicia y ejecuta tu cliente BitTorrent
   ▼

7. CLIENTE BITTORRENT (src/client/)
   │ - Lee el .torrent
   │ - Se conecta al overlay network (gossip)
   │ - Descubre peers (seeder)
   │ - Inicia descarga de piezas
   │ - Logs: "[INFO] Connected to seeder..."
   ▼

8. DOCKER CLIENT
   │ return containerID al handler
   ▼

9. HANDLER
   │ c.JSON(200, gin.H{
   │   "success": true,
   │   "containerId": containerID,
   │   "message": "Container created successfully"
   │ })
   ▼

10. FRONTEND
    │ Recibe respuesta JSON
    │ Actualiza UI: "✅ Leecher1 created"
    │ Refresca lista de contenedores
    └─ FIN
```

---

## 🔌 **Endpoints del API**

### **REST API (HTTP)**

| Endpoint | Método | Función | Parámetros |
|----------|--------|---------|------------|
| `/health` | GET | Health check | - |
| `/api/containers` | GET | Lista todos los contenedores | - |
| `/api/containers` | POST | Crea nuevo contenedor | Body: ContainerRequest |
| `/api/containers/:id` | GET | Info de un contenedor | Path: id |
| `/api/containers/:id/start` | POST | Inicia contenedor | Path: id |
| `/api/containers/:id/stop` | POST | Detiene contenedor | Path: id |
| `/api/containers/:id/restart` | POST | Reinicia contenedor | Path: id |
| `/api/containers/:id` | DELETE | Elimina contenedor | Path: id, Query: force |
| `/api/containers/:id/logs` | GET | Obtiene logs | Path: id, Query: tail |
| `/api/containers/:id/stats` | GET | Estadísticas | Path: id |
| `/api/torrents` | GET | Lista .torrent | - |
| `/api/torrents/upload` | POST | Sube .torrent | Form: file |
| `/api/torrents/:name` | DELETE | Elimina .torrent | Path: name |
| `/api/networks` | GET | Lista redes Docker | - |
| `/api/networks` | POST | Crea red | Body: {name, driver} |

### **WebSocket**

| Endpoint | Función | Datos enviados |
|----------|---------|----------------|
| `/ws/logs/:id` | Stream de logs en tiempo real | Líneas de log (text) |

---

## 🧩 **Componentes del Backend**

### **1. main.go - Servidor HTTP**

**Responsabilidades:**
- Iniciar servidor Gin en puerto 8090
- Configurar CORS para permitir frontend
- Definir todas las rutas (endpoints)
- Upgrade HTTP → WebSocket

**Tecnologías:**
- `gin-gonic/gin` - Framework web
- `gin-contrib/cors` - Middleware CORS

---

### **2. docker/client.go - Docker SDK Wrapper**

**Responsabilidades:**
- Envolver funciones del Docker SDK
- Crear/iniciar/detener/eliminar contenedores
- Obtener logs y estadísticas
- Gestionar redes Docker

**Funciones principales:**
```go
NewDockerClient() (*DockerClient, error)
ListContainers() ([]types.Container, error)
CreateContainer(config CreateContainerConfig) (string, error)
StartContainer(containerID string) error
StopContainer(containerID string) error
RestartContainer(containerID string) error
RemoveContainer(containerID string, force bool) error
GetLogs(containerID string, tail string) (string, error)
StreamLogs(containerID string) (io.ReadCloser, error)
GetStats(containerID string) (types.StatsJSON, error)
ListNetworks() ([]types.NetworkResource, error)
CreateNetwork(name string, driver string) (string, error)
```

**Tecnologías:**
- `docker/docker/client` - Docker SDK oficial

---

### **3. handlers/containers.go - Endpoints de contenedores**

**Responsabilidades:**
- Recibir peticiones HTTP del frontend
- Validar datos de entrada
- Llamar funciones del DockerClient
- Formatear respuestas JSON

**Handlers:**
```go
ListContainers(c *gin.Context)
GetContainer(c *gin.Context)
CreateContainer(c *gin.Context)  ← Más complejo
StartContainer(c *gin.Context)
StopContainer(c *gin.Context)
RestartContainer(c *gin.Context)
DeleteContainer(c *gin.Context)
GetLogs(c *gin.Context)
GetStats(c *gin.Context)
ListNetworks(c *gin.Context)
CreateNetwork(c *gin.Context)
```

---

### **4. handlers/torrents.go - Gestión de .torrent**

**Responsabilidades:**
- Listar archivos .torrent en `archives/torrents/`
- Subir nuevos archivos desde el frontend
- Eliminar archivos .torrent

**Handlers:**
```go
ListTorrents(c *gin.Context)
UploadTorrent(c *gin.Context)   ← Multipart form-data
DeleteTorrent(c *gin.Context)
```

---

### **5. handlers/logs.go - WebSocket streaming**

**Responsabilidades:**
- Upgrade HTTP → WebSocket
- Stream de logs en tiempo real
- Mantener conexión viva (heartbeat)
- Detectar desconexiones

**Handler:**
```go
StreamLogs(c *gin.Context)
```

**Tecnologías:**
- `gorilla/websocket` - WebSocket library

**Flujo WebSocket:**
```
1. Cliente conecta: ws://localhost:8090/ws/logs/leecher1
2. Servidor hace upgrade de HTTP a WebSocket
3. Servidor obtiene stream de logs desde Docker
4. Servidor lee logs línea por línea
5. Cada línea se envía al cliente vía WebSocket
6. Heartbeat cada 30s para mantener conexión
7. Si cliente desconecta, se cierra stream
```

---

## 🎨 **Componentes del Frontend**

### **1. app/page.tsx - Página principal**

**Responsabilidades:**
- Layout principal
- Tabs para cambiar entre vistas
- Header con logo

**Estado:**
```tsx
const [activeTab, setActiveTab] = useState<'torrents' | 'add'>('torrents')
```

---

### **2. components/add-torrent-form.tsx**

**Responsabilidades:**
- Formulario para crear contenedores
- Validación de inputs
- Generar request JSON
- Enviar POST al backend

**Campos del formulario:**
- Container Name
- Network Name
- Folder Path (volumen local)
- Image Name
- Torrent File (subir archivo)
- Discovery Mode (tracker/overlay)
- Port (para overlay)
- Bootstrap (peer inicial)

**Flujo:**
```tsx
handleSubmit() {
  // 1. Construir JSON con configuración
  const config = { containerName, networkName, ... }
  
  // 2. Enviar al backend
  fetch('http://localhost:8090/api/containers', {
    method: 'POST',
    body: JSON.stringify(config)
  })
  
  // 3. Manejar respuesta
  .then(response => {
    if (response.success) {
      alert('Container created!')
    }
  })
}
```

---

### **3. components/torrent-list.tsx**

**Responsabilidades:**
- Listar contenedores activos
- Mostrar estado (running/stopped)
- Botones de acción (start/stop/delete)
- Ver logs (modal o panel)

**Actualmente:** Usa datos mock, necesitas conectarlo al API:

```tsx
// Cambiar de:
const [torrents] = useState<TorrentItem[]>(mockTorrents)

// A:
const [torrents, setTorrents] = useState<TorrentItem[]>([])

useEffect(() => {
  fetch('http://localhost:8090/api/containers')
    .then(res => res.json())
    .then(data => setTorrents(data))
}, [])
```

---

### **4. components/stats-overview.tsx**

**Responsabilidades:**
- Mostrar estadísticas globales
- Total de contenedores
- Download/Upload speed
- Peers conectados

---

## 🔐 **Seguridad**

### **CORS (Cross-Origin Resource Sharing)**

El backend permite peticiones desde:
- `http://localhost:3000` (frontend dev)
- `http://localhost:3001` (alternativo)

```go
// main.go
cors.New(cors.Config{
    AllowOrigins: []string{
        "http://localhost:3000",
        "http://localhost:3001",
    },
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
})
```

⚠️ **Para producción:**
- Cambiar a dominios específicos
- Agregar autenticación (JWT, API keys)
- Usar HTTPS
- Validar todas las entradas

---

### **Permisos Docker**

El API necesita acceso al socket de Docker:
- `/var/run/docker.sock` (Unix)
- `npipe:////./pipe/docker_engine` (Windows)

**Solución para desarrollo:**
```bash
sudo usermod -aG docker $USER
newgrp docker
```

---

## 🐛 **Debugging**

### **Ver logs del Backend:**

```bash
cd api
go run main.go
# Verás cada petición HTTP y operación Docker
```

### **Ver logs del Frontend:**

```bash
cd web
pnpm dev
# Verás compilación y errores
```

### **Ver logs de contenedores:**

**Desde el navegador:** Botón "View Logs" en la UI

**Desde terminal:**
```bash
docker logs -f leecher1
```

**Desde API:**
```bash
curl http://localhost:8090/api/containers/leecher1/logs?tail=50
```

---

## 📈 **Escalabilidad**

### **Agregar más endpoints:**

1. Definir función en `handlers/`:
```go
func MyNewHandler(c *gin.Context) {
    // Lógica
    c.JSON(200, gin.H{"data": "..."})
}
```

2. Registrar ruta en `main.go`:
```go
api.GET("/my-new-endpoint", handlers.MyNewHandler)
```

3. Llamar desde frontend:
```tsx
fetch('http://localhost:8090/api/my-new-endpoint')
```

---

### **Agregar más funciones Docker:**

1. Agregar método en `docker/client.go`:
```go
func (dc *DockerClient) MyNewFunction() error {
    // Usar dc.cli...
}
```

2. Llamar desde handler:
```go
func MyHandler(c *gin.Context) {
    dockerClient.MyNewFunction()
}
```

---

## 🎯 **Mejoras futuras sugeridas**

✨ **Backend:**
- [ ] Autenticación JWT
- [ ] Rate limiting
- [ ] Caching de datos
- [ ] Métricas con Prometheus
- [ ] Logs estructurados (JSON)

✨ **Frontend:**
- [ ] Conectar con datos reales (quitar mocks)
- [ ] Gráficos de progreso en tiempo real
- [ ] Visualización de topología P2P
- [ ] Notificaciones push
- [ ] Dark mode
- [ ] Responsive design mejorado

✨ **Docker:**
- [ ] Docker Compose para levantar todo
- [ ] Healthchecks en contenedores
- [ ] Resource limits (CPU, memoria)
- [ ] Auto-restart policies

---

¡Ahora tienes una comprensión completa de cómo funciona todo el sistema! 🚀
