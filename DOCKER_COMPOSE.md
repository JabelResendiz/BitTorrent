# 🐳 Docker Compose - Frontend + Backend

Solución con **Docker Compose** que separa Frontend y Backend en contenedores independientes conectados por la red `net`.

## 📋 Requisitos

- Docker y Docker Compose instalados
- Red Docker `net` (se crea automáticamente si no existe)

## 🚀 Uso Rápido

### Opción 1: Todo en un comando (Recomendado)

```bash
./scripts/setup_compose.sh
```

Este script:
- ✅ Crea la red `net` si no existe
- ✅ Construye ambas imágenes (backend y frontend)
- ✅ Inicia los contenedores
- ✅ Muestra el estado final

### Opción 2: Paso a paso

```bash
# 1. Construir imágenes
./scripts/build_compose.sh

# 2. Ejecutar contenedores
./scripts/run_compose.sh

# 3. Detener contenedores
./scripts/stop_compose.sh
```

## 🌐 Acceso a servicios

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:7000
- **Health Check**: http://localhost:7000/health

## 📊 Comandos útiles

```bash
# Ver logs en tiempo real
docker-compose logs -f

# Ver logs solo del backend
docker-compose logs -f backend

# Ver logs solo del frontend
docker-compose logs -f frontend

# Ver estado de contenedores
docker-compose ps

# Reiniciar servicios
docker-compose restart

# Reconstruir y reiniciar
docker-compose up -d --build

# Detener sin eliminar
docker-compose stop

# Detener y eliminar
docker-compose down

# Eliminar todo incluyendo volúmenes
docker-compose down -v
```

## 🔧 Estructura de servicios

```yaml
services:
  backend:
    - Puerto: 7000
    - Healthcheck: /health
    - Acceso: Docker socket
    
  frontend:
    - Puerto: 3000
    - Depende de: backend
    - API URL: http://localhost:7000
```

Ambos servicios están en la red `net` y pueden comunicarse entre sí.

## 🌍 Uso en otra computadora

### Opción 1: Clonar repositorio y construir

```bash
git clone <tu-repo>
cd BitTorrent
./scripts/setup_compose.sh
```

### Opción 2: Exportar imágenes

**En tu computadora:**
```bash
# Construir imágenes
./scripts/build_compose.sh

# Exportar ambas imágenes
docker save bittorrent-backend bittorrent-frontend | gzip > bittorrent-stack.tar.gz
```

**En otra computadora:**
```bash
# Importar imágenes
docker load < bittorrent-stack.tar.gz

# Crear red
docker network create net

# Copiar docker-compose.yml y ejecutar
docker-compose up -d
```

### Opción 3: Docker Registry (Producción)

**Subir imágenes:**
```bash
# Tag imágenes
docker tag bittorrent-backend tuusuario/bittorrent-backend:latest
docker tag bittorrent-frontend tuusuario/bittorrent-frontend:latest

# Push a Docker Hub
docker push tuusuario/bittorrent-backend:latest
docker push tuusuario/bittorrent-frontend:latest
```

**Descargar en otra máquina:**
```bash
# Actualizar docker-compose.yml
# Cambiar build por image:
#   image: tuusuario/bittorrent-backend:latest
#   image: tuusuario/bittorrent-frontend:latest

docker-compose pull
docker-compose up -d
```

## 🔍 Troubleshooting

### La red 'net' no existe
```bash
docker network create net
```

### Puertos ya en uso
Edita `docker-compose.yml`:
```yaml
ports:
  - "3001:3000"  # Frontend en 3001
  - "7001:7000"  # Backend en 7001
```

### Backend no se conecta a Docker
Verifica permisos del socket:
```bash
sudo usermod -aG docker $USER
# Cerrar sesión y volver a entrar
```

### Frontend no se conecta al Backend
Verifica la variable de entorno en `docker-compose.yml`:
```yaml
environment:
  - NEXT_PUBLIC_API_URL=http://localhost:7000
```

### Error al construir
```bash
# Limpiar cache y reconstruir
docker-compose build --no-cache
```

### Contenedores no inician
```bash
# Ver logs detallados
docker-compose logs

# Verificar salud del backend
docker-compose exec backend wget -O- http://localhost:7000/health
```

## 🔄 Actualizar código

Después de hacer cambios en el código:

```bash
# Opción 1: Reconstruir y reiniciar
docker-compose up -d --build

# Opción 2: Reconstruir específico
docker-compose build backend
docker-compose up -d backend

# Opción 3: Reconstruir todo desde cero
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## 🆚 Ventajas vs Fullstack

| Aspecto | Docker Compose | Fullstack |
|---------|----------------|-----------|
| Separación | ✅ Servicios independientes | ❌ Todo junto |
| Escalabilidad | ✅ Escalar por servicio | ❌ Todo o nada |
| Desarrollo | ✅ Rebuild parcial | ❌ Rebuild completo |
| Logs | ✅ Por servicio | ⚠️ Mezclados |
| Networking | ✅ Interno Docker | ⚠️ Localhost |
| Producción | ✅ Recomendado | ⚠️ No ideal |

## 📝 Notas

- ✅ El frontend espera a que el backend esté saludable antes de iniciar
- ✅ Ambos servicios se reinician automáticamente si fallan
- ✅ Logs separados por servicio
- ✅ Fácil de escalar horizontalmente
- ⚠️ Requiere Docker Compose instalado
- ⚠️ La red `net` debe existir (se crea automáticamente)
