# 🚀 Quick Start - BitTorrent Web Interface

Guía rápida para levantar el sistema completo: Frontend + Backend API + Docker

---

## 📋 **Requisitos previos**

✅ **Docker** instalado y corriendo  
✅ **Go 1.21+** instalado  
✅ **Node.js 18+** y **pnpm** instalados  
✅ Imagen Docker `client_img` construida (tu cliente BitTorrent)

---

## 🎯 **Pasos para iniciar todo**

### **1️⃣ Construir la imagen Docker del cliente (si no lo hiciste aún)**

```bash
cd src
docker build -t client_img -f client/Dockerfile .
```

### **2️⃣ Crear la red Docker**

```bash
docker network create bittorrent
```

### **3️⃣ Iniciar el Backend API**

**Opción A: Con el script (recomendado)**
```bash
cd api
./start.sh
```

**Opción B: Manualmente**
```bash
cd api
go run main.go
```

Deberías ver:
```
✅ Docker client initialized successfully
🚀 BitTorrent API Server starting on http://localhost:8090
📡 WebSocket available at ws://localhost:8090/ws/logs/:id
🌐 Accepting requests from http://localhost:3000
```

### **4️⃣ Iniciar el Frontend**

En otra terminal:

```bash
cd web
pnpm install   # Solo la primera vez
pnpm dev
```

Deberías ver:
```
▲ Next.js 16.0.3
- Local:        http://localhost:3000
```

### **5️⃣ Abrir el navegador**

Visita: **http://localhost:3000**

---

## 🎮 **Cómo usar la interfaz**

### **📤 Subir un torrent**

1. Ve a la pestaña **"Add New Torrent"**
2. Sube tu archivo `.torrent`
3. Configura los parámetros:
   - **Container Name**: `seeder` o `leecher1`
   - **Network Name**: `bittorrent`
   - **Folder Path**: `/home/user/archives/seeder` (ruta local)
   - **Discovery Mode**: `overlay` o `tracker`
   - **Port**: `6000` (para overlay)
   - **Bootstrap**: vacío para el primer nodo
4. Click en **Submit**

### **📊 Ver contenedores activos**

1. Ve a la pestaña **"Active Torrents"**
2. Verás la lista de contenedores corriendo
3. Puedes:
   - Ver logs en tiempo real
   - Detener/Iniciar contenedores
   - Ver estadísticas

---

## 🧪 **Prueba rápida (con curl)**

### **Verificar que el API funciona:**

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

---

## 🐛 **Troubleshooting**

### **Error: "Cannot connect to Docker daemon"**

```bash
# Verificar que Docker está corriendo
docker ps

# Si no está corriendo
sudo systemctl start docker

# Agregar tu usuario al grupo docker (para no usar sudo)
sudo usermod -aG docker $USER
newgrp docker
```

### **Error: "Port 8090 already in use"**

```bash
# Ver qué proceso usa el puerto
lsof -i :8090

# Matar el proceso
kill -9 <PID>
```

### **Error: "ECONNREFUSED" en el frontend**

- Verifica que el backend API esté corriendo en puerto 8090
- Verifica los logs del API en la terminal

### **Error: "Network bittorrent not found"**

```bash
# Crear la red
docker network create bittorrent
```

---

## 📂 **Estructura de archivos importante**

```
BitTorrent/
├── api/                    ← Backend (puerto 8090)
│   ├── main.go
│   ├── start.sh           ← Script para iniciar
│   └── ...
│
├── web/                    ← Frontend (puerto 3000)
│   ├── package.json
│   └── ...
│
├── src/                    ← Tu cliente BitTorrent
│   ├── client/
│   │   └── Dockerfile     ← Para construir client_img
│   └── ...
│
└── archives/
    ├── torrents/          ← Aquí se suben los .torrent
    ├── seeder/            ← Archivos del seeder
    ├── leecher1/          ← Archivos del leecher1
    └── leecher2/          ← Archivos del leecher2
```

---

## 🎯 **Flujo completo de ejemplo**

### **Escenario: Compartir un video**

1. **Preparar el archivo:**
   ```bash
   cp mi-video.mp4 archives/seeder/
   ```

2. **Crear el .torrent:**
   ```bash
   mktorrent -a http://tracker:8090/announce \
     -o archives/torrents/video.torrent \
     archives/seeder/mi-video.mp4
   ```

3. **Iniciar Backend y Frontend** (pasos 3 y 4 arriba)

4. **Crear seeder desde la UI:**
   - Container Name: `seeder`
   - Folder Path: `/home/user/archives/seeder` (ajusta tu ruta)
   - Torrent: `video.torrent`
   - Port: `6000`
   - Discovery Mode: `overlay`
   - Bootstrap: (vacío)

5. **Crear leecher desde la UI:**
   - Container Name: `leecher1`
   - Folder Path: `/home/user/archives/leecher1`
   - Torrent: `video.torrent`
   - Port: `6001`
   - Bootstrap: `seeder:6000`

6. **Ver logs en tiempo real** desde la UI

---

## 🚪 **Detener todo**

### **Detener el Frontend:**
```bash
# En la terminal donde corre Next.js
Ctrl + C
```

### **Detener el Backend:**
```bash
# En la terminal donde corre el API
Ctrl + C
```

### **Detener contenedores:**
```bash
# Desde la UI (botón Stop) o manualmente:
docker stop seeder leecher1 leecher2
docker rm seeder leecher1 leecher2
```

---

## 📖 **Documentación adicional**

- **Backend API completo:** Ver `api/README.md`
- **Arquitectura BitTorrent:** Ver `Documentation/ARQUITECTURA_P2P.md`
- **Overlay Gossip:** Ver `Documentation/OVERLAY_GOSSIP_IMPLEMENTATION.md`

---

## ✨ **Tips**

💡 **Persistir datos:** Los contenedores usan volúmenes en `archives/`, los datos persisten después de detener contenedores

💡 **Ver logs del API:** Se muestran en tiempo real en la terminal donde corre `go run main.go`

💡 **Desarrollo del frontend:** Next.js recarga automáticamente al editar archivos

💡 **Probar sin UI:** Usa `curl` o Postman para probar endpoints directamente

---

¡Listo! Ahora tienes una interfaz web completa para tu sistema BitTorrent distribuido 🎉
