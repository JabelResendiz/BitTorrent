package tracker

// tracker/sync.go
// Sistema de sincronización distribuida entre trackers usando gossip push.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// SyncManager maneja la sincronización periódica con otros trackers (push).
type SyncManager struct {
	tracker      *Tracker
	remotePeers  []string      // Direcciones de otros trackers
	syncInterval time.Duration // Intervalo entre sincronizaciones
	stopCh       chan struct{}
}

// SyncListener escucha conexiones entrantes de otros trackers para recibir sincronización.
type SyncListener struct {
	tracker  *Tracker
	listener net.Listener
	stopCh   chan struct{}
}

// NewSyncManager crea un nuevo gestor de sincronización.
func NewSyncManager(tracker *Tracker, remotePeers []string, syncInterval time.Duration) *SyncManager {
	return &SyncManager{
		tracker:      tracker,
		remotePeers:  remotePeers,
		syncInterval: syncInterval,
		stopCh:       make(chan struct{}),
	}
}

// Start inicia el proceso de sincronización periódica (push a otros trackers).
func (sm *SyncManager) Start() {
	log.Printf("[SYNC] Starting sync manager with %d remote peers, interval=%v", len(sm.remotePeers), sm.syncInterval)

	go func() {
		ticker := time.NewTicker(sm.syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.pushToAllPeers()
			case <-sm.stopCh:
				log.Println("[SYNC] Sync manager stopped")
				return
			}
		}
	}()
}

// Stop detiene el proceso de sincronización.
func (sm *SyncManager) Stop() {
	close(sm.stopCh)
}

// pushToAllPeers envía el estado actual a todos los peers remotos.
func (sm *SyncManager) pushToAllPeers() {
	msg := sm.tracker.NewSyncMessage()

	log.Printf("[SYNC] Pushing state to %d peers (swarms=%d)", len(sm.remotePeers), len(msg.Swarms))

	for _, remotePeer := range sm.remotePeers {
		go sm.pushToPeer(remotePeer, msg)
	}
}

// pushToPeer envía un mensaje de sincronización a un peer específico.
func (sm *SyncManager) pushToPeer(remotePeer string, msg *SyncMessage) {
	url := fmt.Sprintf("http://%s/sync", remotePeer)

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[SYNC] Error marshaling sync message for %s: %v", remotePeer, err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("[SYNC] Error pushing to %s: %v", remotePeer, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[SYNC] Push to %s failed with status %d", remotePeer, resp.StatusCode)
		return
	}

	log.Printf("[SYNC] Successfully pushed to %s", remotePeer)
}

// NewSyncListener crea un nuevo listener de sincronización.
func NewSyncListener(tracker *Tracker, listenAddr string) (*SyncListener, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}

	return &SyncListener{
		tracker:  tracker,
		listener: listener,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start inicia el servidor de sincronización para recibir mensajes de otros trackers.
func (sl *SyncListener) Start() {
	log.Printf("[SYNC] Sync listener started on %s", sl.listener.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/sync", sl.handleSync)

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		<-sl.stopCh
		log.Println("[SYNC] Shutting down sync listener")
		sl.listener.Close()
	}()

	go func() {
		if err := server.Serve(sl.listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[SYNC] Sync listener error: %v", err)
		}
	}()
}

// Stop detiene el listener de sincronización.
func (sl *SyncListener) Stop() {
	close(sl.stopCh)
}

// handleSync maneja las peticiones de sincronización entrantes.
func (sl *SyncListener) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[SYNC] Error reading sync request: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var msg SyncMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("[SYNC] Error unmarshaling sync message: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("[SYNC] Received sync from node %s with %d swarms", msg.FromNodeID, len(msg.Swarms))

	// Hacer merge del estado recibido
	sl.tracker.MergeSwarms(&msg)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
