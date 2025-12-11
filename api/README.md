# 🚀 BitTorrent API - Backend con Docker SDK

API REST construida en **Go** que actúa como puente entre el frontend web y Docker Engine, permitiendo gestionar contenedores BitTorrent desde una interfaz gráfica.

---

## 📋 **Características**

✅ **Gestión completa de contenedores Docker**
- Crear y lanzar contenedores BitTorrent
- Listar, iniciar, detener, reiniciar y eliminar contenedores
- Obtener logs en tiempo real mediante WebSocket
- Estadísticas de uso (CPU, memoria, red)

✅ **Gestión de archivos .torrent**
- Listar archivos .torrent disponibles
- Subir nuevos archivos .torrent
- Eliminar archivos .torrent

✅ **Gestión de redes Docker**
- Listar redes existentes
- Crear nuevas redes overlay

---

## 🏗️ **Arquitectura**

```
┌─────────────────────────────────────────┐
│     FRONTEND (Next.js - Puerto 3000)    │
└──────────────┬──────────────────────────┘
               │ HTTP/WebSocket
               ▼
┌─────────────────────────────────────────┐
│      API (Go - Puerto 8090)             │
│   ├── main.go                           │
│   ├── handlers/                         │
│   │   ├── containers.go                 │
│   │   ├── torrents.go                   │
│   │   └── logs.go                       │
│   └── docker/                           │
│       └── client.go (Docker SDK)        │
└──────────────┬──────────────────────────┘
               │ Docker API
               ▼
┌─────────────────────────────────────────┐
│        DOCKER ENGINE                     │
│   - Contenedores BitTorrent             │
│   - Networks (overlay)                  │
│   - Volumes                             │
└─────────────────────────────────────────┘
```

---

## 📦 **Instalación**

### **1. Requisitos previos**

- Go 1.21 o superior
- Docker Engine instalado y corriendo
- Usuario con permisos para acceder al socket de Docker

### **2. Clonar el repositorio**

```bash
cd BitTorrent/api
```

### **3. Descargar dependencias**

```bash
go mod download
```

---

## 🚀 **Uso**

### **Iniciar el servidor API**

```bash
go run main.go
```

Salida esperada:
```
✅ Docker client initialized successfully
🚀 BitTorrent API Server starting on http://localhost:8090
📡 WebSocket available at ws://localhost:8090/ws/logs/:id
🌐 Accepting requests from http://localhost:3000
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[GIN-debug] Listening and serving HTTP on :8090
```

### **Verificar que está corriendo**

```bash
curl http://localhost:8090/health
```

Respuesta:
```json
{
  "status": "ok",
  "service": "BitTorrent API",
  "version": "1.0.0"
}
```

---

## 📡 **Endpoints disponibles**

### **🐳 Contenedores**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| `GET` | `/api/containers` | Lista todos los contenedores |
| `POST` | `/api/containers` | Crea y arranca un nuevo contenedor |
| `GET` | `/api/containers/:id` | Obtiene información de un contenedor |
| `POST` | `/api/containers/:id/start` | Inicia un contenedor |
| `POST` | `/api/containers/:id/stop` | Detiene un contenedor |
| `POST` | `/api/containers/:id/restart` | Reinicia un contenedor |
| `DELETE` | `/api/containers/:id` | Elimina un contenedor |
| `GET` | `/api/containers/:id/logs` | Obtiene logs de un contenedor |
| `GET` | `/api/containers/:id/stats` | Obtiene estadísticas (CPU, RAM) |

### **📄 Torrents**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| `GET` | `/api/torrents` | Lista archivos .torrent |
| `POST` | `/api/torrents/upload` | Sube un nuevo .torrent |
| `DELETE` | `/api/torrents/:name` | Elimina un .torrent |

### **🌐 Redes**

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| `GET` | `/api/networks` | Lista redes Docker |
| `POST` | `/api/networks` | Crea una nueva red |

### **📡 WebSocket**

| Endpoint | Descripción |
|----------|-------------|
| `WS /ws/logs/:id` | Stream de logs en tiempo real |

---

## 🔧 **Ejemplos de uso**

### **1. Listar contenedores**

```bash
curl http://localhost:8090/api/containers
```

**Respuesta:**
```json
[
  {
    "id": "a1b2c3d4e5f6",
    "name": "seeder",
    "image": "client_img",
    "state": "running",
    "status": "Up 5 minutes",
    "created": 1702234567,
    "ports": [...]
  }
]
```

### **2. Crear contenedor**

```bash
curl -X POST http://localhost:8090/api/containers \
  -H "Content-Type: application/json" \
  -d '{
    "containerName": "leecher1",
    "networkName": "bittorrent",
    "folderPath": "/home/user/archives/leecher1",
    "imageName": "client_img",
    "torrentFile": "video.torrent",
    "discoveryMode": "overlay",
    "port": "6001",
    "bootstrap": "seeder:6000"
  }'
```

**Respuesta:**
```json
{
  "success": true,
  "containerId": "x9y8z7w6v5u4",
  "name": "leecher1",
  "message": "Container created and started successfully"
}
```

### **3. Obtener logs**

```bash
curl "http://localhost:8090/api/containers/leecher1/logs?tail=50"
```

**Respuesta:**
```json
{
  "logs": "[INFO] Starting client...\n[INFO] Connected to tracker...\n..."
}
```

### **4. Subir torrent**

```bash
curl -X POST http://localhost:8090/api/torrents/upload \
  -F "file=@/path/to/video.torrent"
```

**Respuesta:**
```json
{
  "success": true,
  "filename": "video.torrent",
  "size": 45678,
  "path": "../archives/torrents/video.torrent",
  "message": "Torrent uploaded successfully"
}
```

### **5. WebSocket para logs en tiempo real**

```javascript
// Desde el frontend (JavaScript)
const ws = new WebSocket('ws://localhost:8090/ws/logs/leecher1');

ws.onmessage = (event) => {
  console.log('Log:', event.data);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

---

## 🧪 **Testing**

### **Probar con curl**

```bash
# Health check
curl http://localhost:8090/health

# Listar contenedores
curl http://localhost:8090/api/containers

# Listar torrents
curl http://localhost:8090/api/torrents

# Listar redes
curl http://localhost:8090/api/networks
```

### **Probar con Postman/Thunder Client**

Importa esta colección:

```json
{
  "name": "BitTorrent API",
  "requests": [
    {
      "name": "Health Check",
      "method": "GET",
      "url": "http://localhost:8090/health"
    },
    {
      "name": "List Containers",
      "method": "GET",
      "url": "http://localhost:8090/api/containers"
    }
  ]
}
```

---

## 🔧 **Configuración**

### **Puerto del servidor**

Por defecto: `8090`

Para cambiar, edita `main.go`:
```go
port := ":8090"  // Cambiar a ":9000" por ejemplo
```

### **CORS**

Por defecto permite conexiones desde `http://localhost:3000` y `http://localhost:3001`.

Para agregar más orígenes, edita `main.go`:
```go
AllowOrigins: []string{
    "http://localhost:3000",
    "http://localhost:3001",
    "http://tu-frontend.com",  // Agregar aquí
},
```

### **Directorio de torrents**

Por defecto: `../archives/torrents`

Para cambiar, edita `handlers/torrents.go`:
```go
torrentsDir := "../archives/torrents"  // Cambiar ruta aquí
```

---

## 🐛 **Troubleshooting**

### **Error: "Cannot connect to Docker daemon"**

**Problema:** El API no puede conectarse a Docker.

**Solución:**
```bash
# Verificar que Docker está corriendo
sudo systemctl status docker

# Verificar permisos
sudo usermod -aG docker $USER
newgrp docker

# O ejecutar con sudo
sudo go run main.go
```

### **Error: "bind: address already in use"**

**Problema:** El puerto 8090 está ocupado.

**Solución:**
```bash
# Ver qué proceso usa el puerto
lsof -i :8090

# Matar el proceso
kill -9 <PID>

# O cambiar el puerto en main.go
```

### **Error de CORS en el frontend**

**Problema:** El navegador bloquea peticiones.

**Solución:** Verificar que el origen del frontend esté en la lista de `AllowOrigins` en `main.go`.

---

## 📂 **Estructura del proyecto**

```
api/
├── main.go                 # Punto de entrada, servidor HTTP
├── go.mod                  # Dependencias
├── go.sum                  # Checksums de dependencias
├── docker/
│   └── client.go          # Wrapper del Docker SDK
└── handlers/
    ├── containers.go      # Endpoints de contenedores
    ├── torrents.go        # Endpoints de torrents
    └── logs.go            # WebSocket para logs
```

---

## 📚 **Dependencias**

- **Gin** (`github.com/gin-gonic/gin`) - Framework web
- **Docker SDK** (`github.com/docker/docker/client`) - Cliente de Docker
- **Gorilla WebSocket** (`github.com/gorilla/websocket`) - WebSockets
- **CORS** (`github.com/gin-contrib/cors`) - Middleware para CORS

---

## 🚀 **Despliegue en producción**

### **1. Compilar el binario**

```bash
go build -o bittorrent-api main.go
```

### **2. Ejecutar el binario**

```bash
./bittorrent-api
```

### **3. Como servicio systemd**

Crear `/etc/systemd/system/bittorrent-api.service`:

```ini
[Unit]
Description=BitTorrent API Server
After=docker.service

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/api
ExecStart=/path/to/api/bittorrent-api
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Habilitar y arrancar:
```bash
sudo systemctl daemon-reload
sudo systemctl enable bittorrent-api
sudo systemctl start bittorrent-api
```

---

## 🤝 **Integración con el frontend**

El frontend en `web/` debe configurar la URL de la API:

```typescript
// web/src/services/api.ts
const API_BASE_URL = 'http://localhost:8090/api';

export async function createContainer(config: any) {
  const response = await fetch(`${API_BASE_URL}/containers`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  return response.json();
}
```

---

## 📝 **Notas importantes**

⚠️ **Seguridad:** Esta API está diseñada para desarrollo local. Para producción:
- Agregar autenticación (JWT, API keys)
- Limitar CORS a dominios específicos
- Usar HTTPS
- Validar todas las entradas

⚠️ **Permisos:** El usuario que ejecuta la API necesita acceso al socket de Docker (`/var/run/docker.sock`).

⚠️ **Rutas:** Las rutas de archivos (torrents, volúmenes) son relativas. Ajustar según tu estructura.

---

## 📞 **Soporte**

Si tienes problemas:

1. Verifica que Docker está corriendo: `docker ps`
2. Verifica logs del API en la terminal
3. Prueba endpoints con `curl` para descartar problemas del frontend
4. Revisa los logs de Docker: `docker logs <container-id>`

---

## 📄 **Licencia**

Este proyecto es parte del sistema BitTorrent distribuido para la asignatura de Sistemas Distribuidos 2025.

---

¡Tu API está lista para gestionar contenedores BitTorrent desde el navegador! 🎉
