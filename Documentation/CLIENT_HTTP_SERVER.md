# Cliente HTTP Server - Implementación para Métricas y Control

## Descripción General

Para que la interfaz gráfica pueda mostrar el progreso de descarga, peers conectados, y controlar las descargas (pausar/reanudar), cada contenedor de cliente BitTorrent debe exponer un servidor HTTP interno en el puerto 9091.

## Arquitectura

```
Frontend (puerto 3000)
    ↓
Backend API (puerto 8090)
    ↓
Docker Container (puerto 9091) ← HTTP Server en cliente BitTorrent
```

## Implementación

### 1. Crear archivo `src/client/http_server.go`

```go
package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// StatusResponse contiene las métricas del cliente
type StatusResponse struct {
	State           string  `json:"state"`            // "downloading", "seeding", "completed", "starting"
	Paused          bool    `json:"paused"`
	Progress        float64 `json:"progress"`         // Porcentaje 0-100
	Downloaded      int64   `json:"downloaded"`       // Bytes descargados
	TotalSize       int64   `json:"total_size"`       // Tamaño total del torrent
	DownloadSpeed   int64   `json:"download_speed"`   // Bytes/segundo
	UploadSpeed     int64   `json:"upload_speed"`     // Bytes/segundo
	ConnectedPeers  int     `json:"connected_peers"`  // Peers conectados actualmente
	TotalPeers      int     `json:"total_peers"`      // Total peers conocidos
	Eta             string  `json:"eta"`              // Tiempo estimado restante
}

// HTTPServer maneja las peticiones HTTP del cliente
type HTTPServer struct {
	client *Client
	server *http.Server
	mu     sync.RWMutex
}

// NewHTTPServer crea un nuevo servidor HTTP
func NewHTTPServer(client *Client, port int) *HTTPServer {
	hs := &HTTPServer{
		client: client,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", hs.handleStatus)
	mux.HandleFunc("/pause", hs.handlePause)
	mux.HandleFunc("/resume", hs.handleResume)
	mux.HandleFunc("/health", hs.handleHealth)

	hs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return hs
}

// Start inicia el servidor HTTP
func (hs *HTTPServer) Start() error {
	return hs.server.ListenAndServe()
}

// handleStatus devuelve el estado actual del cliente
func (hs *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hs.mu.RLock()
	defer hs.mu.RUnlock()

	// Calcular métricas
	status := hs.calculateStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handlePause pausa la descarga
func (hs *HTTPServer) handlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hs.mu.Lock()
	defer hs.mu.Unlock()

	// Pausar el cliente
	hs.client.Pause()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

// handleResume reanuda la descarga
func (hs *HTTPServer) handleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hs.mu.Lock()
	defer hs.mu.Unlock()

	// Reanudar el cliente
	hs.client.Resume()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}

// handleHealth endpoint de salud para Docker
func (hs *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// calculateStatus calcula las métricas actuales
func (hs *HTTPServer) calculateStatus() StatusResponse {
	client := hs.client

	// Obtener información del storage
	downloaded := client.storage.Downloaded()
	totalSize := client.storage.TotalSize()
	progress := float64(0)
	if totalSize > 0 {
		progress = (float64(downloaded) / float64(totalSize)) * 100
	}

	// Calcular velocidades (bytes en último segundo)
	downloadSpeed := client.GetDownloadSpeed()
	uploadSpeed := client.GetUploadSpeed()

	// Contar peers
	connectedPeers := len(client.peers)
	totalPeers := client.GetTotalKnownPeers()

	// Calcular ETA
	eta := "∞"
	if downloadSpeed > 0 {
		remaining := totalSize - downloaded
		secondsLeft := remaining / downloadSpeed
		eta = formatDuration(time.Duration(secondsLeft) * time.Second)
	}

	// Determinar estado
	state := "starting"
	if client.IsPaused() {
		state = "paused"
	} else if downloaded >= totalSize {
		state = "completed"
	} else if downloadSpeed > 0 {
		state = "downloading"
	} else if uploadSpeed > 0 {
		state = "seeding"
	}

	return StatusResponse{
		State:          state,
		Paused:         client.IsPaused(),
		Progress:       progress,
		Downloaded:     downloaded,
		TotalSize:      totalSize,
		DownloadSpeed:  downloadSpeed,
		UploadSpeed:    uploadSpeed,
		ConnectedPeers: connectedPeers,
		TotalPeers:     totalPeers,
		Eta:            eta,
	}
}

// formatDuration formatea duración en formato legible
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "∞"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
```

### 2. Modificar `src/client/client.go`

Agregar campos y métodos necesarios:

```go
type Client struct {
	// ... campos existentes ...
	
	// Nuevos campos para HTTP server
	httpServer *HTTPServer
	paused     bool
	pauseMu    sync.RWMutex
	
	// Métricas
	downloadSpeed int64
	uploadSpeed   int64
	speedMu       sync.RWMutex
}

// Pause pausa la descarga
func (c *Client) Pause() {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()
	c.paused = true
	c.logger.Info("Client paused")
}

// Resume reanuda la descarga
func (c *Client) Resume() {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()
	c.paused = false
	c.logger.Info("Client resumed")
}

// IsPaused devuelve si está pausado
func (c *Client) IsPaused() bool {
	c.pauseMu.RLock()
	defer c.pauseMu.RUnlock()
	return c.paused
}

// GetDownloadSpeed devuelve velocidad de descarga
func (c *Client) GetDownloadSpeed() int64 {
	c.speedMu.RLock()
	defer c.speedMu.RUnlock()
	return c.downloadSpeed
}

// GetUploadSpeed devuelve velocidad de subida
func (c *Client) GetUploadSpeed() int64 {
	c.speedMu.RLock()
	defer c.speedMu.RUnlock()
	return c.uploadSpeed
}

// UpdateDownloadSpeed actualiza velocidad de descarga
func (c *Client) UpdateDownloadSpeed(speed int64) {
	c.speedMu.Lock()
	defer c.speedMu.Unlock()
	c.downloadSpeed = speed
}

// UpdateUploadSpeed actualiza velocidad de subida
func (c *Client) UpdateUploadSpeed(speed int64) {
	c.speedMu.Lock()
	defer c.speedMu.Unlock()
	c.uploadSpeed = speed
}

// GetTotalKnownPeers devuelve total de peers conocidos
func (c *Client) GetTotalKnownPeers() int {
	// Implementar según tu lógica actual
	// Puede ser suma de peers del tracker + overlay
	return len(c.peers) + c.overlay.GetPeerCount()
}
```

### 3. Modificar `src/client/storage.go`

Agregar métodos para obtener métricas:

```go
// Downloaded devuelve bytes totales descargados
func (s *Storage) Downloaded() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	downloaded := int64(0)
	for i := 0; i < len(s.pieces); i++ {
		if s.pieces[i] {
			downloaded += int64(s.pieceLength)
		}
	}
	// Ajustar última pieza si es más pequeña
	if s.pieces[len(s.pieces)-1] {
		lastPieceSize := s.totalSize % int64(s.pieceLength)
		if lastPieceSize > 0 {
			downloaded -= int64(s.pieceLength) - lastPieceSize
		}
	}
	return downloaded
}

// TotalSize devuelve tamaño total del archivo
func (s *Storage) TotalSize() int64 {
	return s.totalSize
}
```

### 4. Modificar `src/client/runtime/runtime_start.go`

Agregar inicio del servidor HTTP:

```go
func (r *Runtime) Start() error {
	// ... código existente para iniciar cliente ...

	// Obtener puerto HTTP desde flag --http-port
	httpPort := r.config.HTTPPort // Asegúrate que Config tenga este campo
	if httpPort == 0 {
		httpPort = 9091 // Puerto por defecto
	}

	// Iniciar servidor HTTP
	httpServer := client.NewHTTPServer(r.client, httpPort)
	r.client.httpServer = httpServer
	
	go func() {
		r.logger.Info(fmt.Sprintf("Starting HTTP server on port %d", httpPort))
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			r.logger.Error("HTTP server error: " + err.Error())
		}
	}()

	// ... resto del código existente ...
	
	return nil
}
```

### 5. Modificar `src/client/config.go`

Agregar campo para el puerto HTTP:

```go
type Config struct {
	// ... campos existentes ...
	HTTPPort int // Puerto para servidor HTTP interno
}
```

### 6. Modificar `src/client/cmd/main.go`

Agregar flag para el puerto HTTP:

```go
var (
	// ... flags existentes ...
	httpPort = flag.Int("http-port", 9091, "HTTP server port for metrics")
)

func main() {
	flag.Parse()
	
	config := &client.Config{
		// ... configuración existente ...
		HTTPPort: *httpPort,
	}
	
	// ... resto del código ...
}
```

## Monitoreo de Velocidad

Para calcular velocidades en tiempo real, implementa un sistema de muestreo:

```go
// En client.go, agregar método para actualizar velocidades periódicamente
func (c *Client) startSpeedMonitoring() {
	ticker := time.NewTicker(1 * time.Second)
	lastDownloaded := c.storage.Downloaded()
	lastUploaded := int64(0) // Necesitas trackear bytes subidos

	go func() {
		for range ticker.C {
			currentDownloaded := c.storage.Downloaded()
			downloadSpeed := currentDownloaded - lastDownloaded
			lastDownloaded = currentDownloaded
			c.UpdateDownloadSpeed(downloadSpeed)

			// Similar para upload
			currentUploaded := c.GetTotalUploaded() // Implementar este método
			uploadSpeed := currentUploaded - lastUploaded
			lastUploaded = currentUploaded
			c.UpdateUploadSpeed(uploadSpeed)
		}
	}()
}
```

## Compilación y Testing

### Compilar el cliente con HTTP server:

```bash
cd src/client
go build -o ../../main ./cmd/main.go
```

### Probar el servidor HTTP:

```bash
# Iniciar un cliente
./main --torrent-file=test.torrent --http-port=9091

# En otra terminal, consultar status
curl http://localhost:9091/status

# Pausar descarga
curl -X POST http://localhost:9091/pause

# Reanudar descarga
curl -X POST http://localhost:9091/resume
```

## Respuesta de Ejemplo

```json
{
  "state": "downloading",
  "paused": false,
  "progress": 45.32,
  "downloaded": 23456789,
  "total_size": 51773349,
  "download_speed": 1048576,
  "upload_speed": 524288,
  "connected_peers": 5,
  "total_peers": 12,
  "eta": "25s"
}
```

## Notas Importantes

1. **Thread Safety**: Todos los métodos que acceden a `client` deben usar mutexes apropiados
2. **Pausar/Reanudar**: Implementar lógica para detener/reiniciar la descarga de bloques
3. **Velocidades**: Actualizar cada segundo para métricas precisas
4. **Puerto 9091**: Debe ser expuesto en el contenedor Docker (ya configurado en el backend)
5. **Error Handling**: Manejar casos donde el storage o peers no están inicializados

## Integración con Backend

El backend API ya está configurado para:
- Crear contenedores con `--http-port=9091` flag
- Exponer puerto 9091 con port bindings
- Proxy de peticiones a `/api/containers/:id/status`, `/pause`, `/resume`

## Próximos Pasos

1. Implementar los archivos descritos arriba
2. Compilar el cliente actualizado
3. Reconstruir la imagen Docker del cliente
4. Probar la integración completa con el frontend
