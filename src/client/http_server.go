package client

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"src/peerwire"
	"sync"
	"time"
)

// StatusResponse contiene las métricas del cliente
type StatusResponse struct {
	TorrentName    string  `json:"torrent_name"` // Nombre del archivo torrent
	State          string  `json:"state"`        // "downloading", "seeding", "completed", "starting"
	Paused         bool    `json:"paused"`
	Progress       float64 `json:"progress"`        // Porcentaje 0-100
	Downloaded     int64   `json:"downloaded"`      // Bytes descargados
	TotalSize      int64   `json:"total_size"`      // Tamaño total del torrent
	DownloadSpeed  int64   `json:"download_speed"`  // Bytes/segundo
	UploadSpeed    int64   `json:"upload_speed"`    // Bytes/segundo
	ConnectedPeers int     `json:"connected_peers"` // Peers conectados actualmente
	TotalPeers     int     `json:"total_peers"`     // Total peers conocidos
	Eta            string  `json:"eta"`             // Tiempo estimado restante
}

// HTTPServer maneja las peticiones HTTP del cliente
type HTTPServer struct {
	store          *peerwire.DiskPieceStore
	manager        *peerwire.Manager
	fileLength     int64
	torrentName    string
	server         *http.Server
	mu             sync.RWMutex
	downloadSpeed  int64
	uploadSpeed    int64
	lastDownloaded int64
	lastUploaded   int64
	stopMonitoring chan struct{}
}

var globalPaused bool
var globalPausedMu sync.RWMutex

// SetGlobalPause establece el estado de pausa global
func SetGlobalPause(paused bool) {
	globalPausedMu.Lock()
	defer globalPausedMu.Unlock()
	globalPaused = paused
}

// IsGlobalPaused retorna si el cliente está pausado globalmente
func IsGlobalPaused() bool {
	globalPausedMu.RLock()
	defer globalPausedMu.RUnlock()
	return globalPaused
}

// NewHTTPServer crea un nuevo servidor HTTP
func NewHTTPServer(store *peerwire.DiskPieceStore, manager *peerwire.Manager, fileLength int64, torrentName string, port int) *HTTPServer {
	hs := &HTTPServer{
		store:          store,
		manager:        manager,
		fileLength:     fileLength,
		torrentName:    torrentName,
		stopMonitoring: make(chan struct{}),
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

	// Iniciar monitoreo de velocidad
	go hs.startSpeedMonitoring()

	return hs
}

// Start inicia el servidor HTTP
func (hs *HTTPServer) Start() error {
	return hs.server.ListenAndServe()
}

// Stop detiene el servidor HTTP
func (hs *HTTPServer) Stop() error {
	close(hs.stopMonitoring)
	if hs.server != nil {
		return hs.server.Close()
	}
	return nil
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(status)
}

// handlePause pausa la descarga
func (hs *HTTPServer) handlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	SetGlobalPause(true)

	fmt.Println("[HTTP] Cliente pausado")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

// handleResume reanuda la descarga
func (hs *HTTPServer) handleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	SetGlobalPause(false)

	fmt.Println("[HTTP] Cliente reanudado")

	// Reactivar descarga en peers disponibles
	go hs.resumeDownload()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}

// resumeDownload reactiva la descarga buscando piezas pendientes
func (hs *HTTPServer) resumeDownload() {
	if hs.manager == nil || hs.store == nil {
		return
	}

	numPieces := hs.store.NumPieces()

	// Intentar encontrar piezas pendientes
	for i := 0; i < numPieces; i++ {
		if !hs.store.HasPiece(i) {
			// Encontramos una pieza que falta, intentar descargarla
			fmt.Printf("[HTTP] Reactivando descarga desde pieza %d\n", i)
			hs.manager.DownloadPieceParallel(i)
			return
		}
	}

	fmt.Println("[HTTP] No hay piezas pendientes para descargar")
}

// handleHealth endpoint de salud para Docker
func (hs *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// IsPaused devuelve si el cliente está pausado
func (hs *HTTPServer) IsPaused() bool {
	return IsGlobalPaused()
}

// calculateStatus calcula las métricas actuales
func (hs *HTTPServer) calculateStatus() StatusResponse {
	// Calcular bytes descargados
	downloaded := hs.getDownloaded()
	progress := float64(0)
	if hs.fileLength > 0 {
		progress = (float64(downloaded) / float64(hs.fileLength)) * 100
		if progress > 100 {
			progress = 100
		}
	}

	// Obtener velocidades
	downloadSpeed := hs.downloadSpeed
	uploadSpeed := hs.uploadSpeed

	// Contar peers conectados
	connectedPeers := hs.manager.GetPeerCount()
	totalPeers := connectedPeers // Por ahora, sin overlay info

	// Calcular ETA
	eta := "∞"
	if downloadSpeed > 0 && downloaded < hs.fileLength {
		remaining := hs.fileLength - downloaded
		secondsLeft := remaining / downloadSpeed
		eta = formatDuration(time.Duration(secondsLeft) * time.Second)
	}

	// Determinar estado
	state := "starting"
	if IsGlobalPaused() {
		state = "paused"
	} else if downloaded >= hs.fileLength {
		state = "completed"
	} else if downloadSpeed > 0 {
		state = "downloading"
	} else if uploadSpeed > 0 {
		state = "seeding"
	}

	return StatusResponse{
		TorrentName:    hs.torrentName,
		State:          state,
		Paused:         IsGlobalPaused(),
		Progress:       progress,
		Downloaded:     downloaded,
		TotalSize:      hs.fileLength,
		DownloadSpeed:  downloadSpeed,
		UploadSpeed:    uploadSpeed,
		ConnectedPeers: connectedPeers,
		TotalPeers:     totalPeers,
		Eta:            eta,
	}
}

// getDownloaded calcula bytes totales descargados
func (hs *HTTPServer) getDownloaded() int64 {
	downloaded := int64(0)
	numPieces := hs.store.NumPieces()
	pieceLength := int64(hs.store.PieceLength())

	for i := 0; i < numPieces; i++ {
		if hs.store.HasPiece(i) {
			if i == numPieces-1 {
				// Última pieza puede ser más pequeña
				lastPieceSize := hs.fileLength % pieceLength
				if lastPieceSize > 0 {
					downloaded += lastPieceSize
				} else {
					downloaded += pieceLength
				}
			} else {
				downloaded += pieceLength
			}
		}
	}

	return downloaded
}

// startSpeedMonitoring monitorea la velocidad de descarga/subida
func (hs *HTTPServer) startSpeedMonitoring() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	hs.lastDownloaded = hs.getDownloaded()
	hs.lastUploaded = 0 // TODO: implementar tracking de upload

	for {
		select {
		case <-ticker.C:
			currentDownloaded := hs.getDownloaded()
			downloadSpeed := currentDownloaded - hs.lastDownloaded
			if downloadSpeed < 0 {
				downloadSpeed = 0
			}

			hs.mu.Lock()
			hs.downloadSpeed = downloadSpeed
			hs.lastDownloaded = currentDownloaded
			// TODO: actualizar uploadSpeed cuando tengamos tracking
			hs.mu.Unlock()

		case <-hs.stopMonitoring:
			return
		}
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

// GetLocalIP obtiene la IP local del contenedor
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
