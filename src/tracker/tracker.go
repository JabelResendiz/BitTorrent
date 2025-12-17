package tracker

//tracker/tracker.go

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Peer: estado mínimo de un peer en un swarm
type Peer struct {
	PeerIDHex string `json:"peer_id"`
	IP        string `json:"ip"`
	Port      uint16 `json:"port"`
	LastSeen  HLC    `json:"last_seen"` // HLC para sincronización distribuida
	Completed bool   `json:"completed"`
	HostName  string `json:"host_name"`
	Deleted   bool   `json:"deleted"` // Tombstone: true si el peer fue eliminado
}

// Swarm: conjunto de peers de un mismo torrent (info_hash)
type Swarm struct {
	Peers map[string]*Peer `json:"peers"` // key: peerIDHex
}

// Tracker: estado global del tracker, configuración y sincronización
type Tracker struct {
	mu           sync.RWMutex
	Torrents     map[string]*Swarm `json:"torrents"` // key: infoHashHex
	Interval     time.Duration     `json:"-"`
	PeerTimeout  time.Duration     `json:"-"`
	MaxPeersResp int               `json:"-"`
	DataPath     string            `json:"-"`

	// Campos para sincronización distribuida
	hlc          HLC           `json:"-"` // Reloj lógico híbrido del tracker
	nodeID       string        `json:"-"` // ID único del tracker
	remotePeers  []string      `json:"-"` // Direcciones de otros trackers
	syncListener *SyncListener `json:"-"` // Servidor de sincronización
	syncManager  *SyncManager  `json:"-"` // Cliente de sincronización
}

// New crea una instancia de Tracker con configuración y estado iniciales.
// interval: segundos que los clientes deben esperar entre announces.
// timeout: tiempo para considerar inactivo a un peer (p. ej., 2×interval).
// maxPeers: máximo de peers a devolver por respuesta.
// dataPath: ruta al archivo JSON para persistencia.
// nodeID: identificador único del tracker para HLC.
// remotePeers: lista de direcciones de otros trackers para sincronización.
func New(interval, timeout time.Duration, maxPeers int, dataPath, nodeID string, remotePeers []string) *Tracker {
	return &Tracker{
		Torrents:     make(map[string]*Swarm),
		Interval:     interval,
		PeerTimeout:  timeout,
		MaxPeersResp: maxPeers,
		DataPath:     dataPath,
		hlc:          *NewHLC(nodeID),
		nodeID:       nodeID,
		remotePeers:  remotePeers,
	}
}

// helper: ensure swarm exists
// getOrCreateSwarm obtiene el swarm asociado a infoHashHex o lo crea si no existe.
func (t *Tracker) getOrCreateSwarm(infoHashHex string) *Swarm {
	sw := t.Torrents[infoHashHex]
	if sw == nil {
		sw = &Swarm{Peers: make(map[string]*Peer)}
		t.Torrents[infoHashHex] = sw
	}
	return sw
}

// Add or update peer
// AddPeer da de alta o actualiza (upsert) un peer dentro del swarm de infoHashHex.
// Actualiza IP, puerto y LastSeen con HLC. Si el peer estaba marcado como eliminado
// (tombstone), lo resucita si esta actualización es más reciente.
func (t *Tracker) AddPeer(infoHashHex, peerIDHex string, hostname string, port uint16, completed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Actualizar HLC para evento local
	t.hlc.Update(nil)

	sw := t.getOrCreateSwarm(infoHashHex)
	p := sw.Peers[peerIDHex]
	if p == nil {
		p = &Peer{PeerIDHex: peerIDHex}
		sw.Peers[peerIDHex] = p
	}

	// Resucitar peer si estaba eliminado (tombstone resurrection)
	if p.Deleted {
		p.Deleted = false
	}

	p.HostName = hostname
	p.IP = hostname // Usar hostname como IP para compatibilidad
	p.Port = port
	p.LastSeen = t.hlc.Clone()
	p.Completed = completed || p.Completed
}

// Remove peer
// RemovePeer marca un peer como eliminado usando tombstone en lugar de borrarlo físicamente.
// Esto permite que la eliminación se propague a otros trackers en el sistema distribuido.
func (t *Tracker) RemovePeer(infoHashHex, peerIDHex string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Actualizar HLC para evento local
	t.hlc.Update(nil)

	sw := t.Torrents[infoHashHex]
	if sw == nil {
		return
	}

	p := sw.Peers[peerIDHex]
	if p == nil {
		return
	}

	// Marcar como eliminado (tombstone) en lugar de borrar
	p.Deleted = true
	p.LastSeen = t.hlc.Clone()
}

// Get peers (excluding one) up to max
// GetPeers devuelve hasta max peers del swarm identificado por infoHashHex,
// excluyendo al peer con ID excludePeerIDHex y peers marcados como eliminados.
// Devuelve copias para no exponer punteros internos.
func (t *Tracker) GetPeers(infoHashHex, excludePeerIDHex string, max int) []*Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	sw := t.Torrents[infoHashHex]
	if sw == nil {
		return nil
	}
	res := make([]*Peer, 0, max)
	for id, p := range sw.Peers {
		if id == excludePeerIDHex || p.Deleted {
			continue
		}
		res = append(res, &Peer{PeerIDHex: p.PeerIDHex, IP: p.IP, HostName: p.HostName, Port: p.Port, LastSeen: p.LastSeen})
		if len(res) >= max {
			break
		}
	}
	return res
}

// Expire old peers
// GC recorre todos los swarms y:
// 1. Marca peers inactivos como eliminados (tombstone) si no lo están ya
// 2. Elimina físicamente tombstones muy antiguos (2x PeerTimeout)
// Devuelve la cantidad de peers procesados.
func (t *Tracker) GC() (expired int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Actualizar HLC para evento local
	t.hlc.Update(nil)

	// Calcular umbrales de tiempo
	thresholdInactive := t.hlc.SubtractDuration(t.PeerTimeout)
	thresholdTombstone := t.hlc.SubtractDuration(2 * t.PeerTimeout)

	for ih, sw := range t.Torrents {
		for id, p := range sw.Peers {
			if p.Deleted {
				// Si el tombstone es muy antiguo, eliminarlo físicamente
				if thresholdTombstone.After(p.LastSeen) {
					delete(sw.Peers, id)
					expired++
				}
			} else {
				// Si el peer está inactivo, marcarlo como eliminado (tombstone)
				if thresholdInactive.After(p.LastSeen) {
					p.Deleted = true
					p.LastSeen = t.hlc.Clone()
					expired++
				}
			}
		}
		if len(sw.Peers) == 0 {
			delete(t.Torrents, ih)
		}
	}
	return
}

// CountPeers retorna el número de seeders (complete) y leechers (incomplete)
// en el swarm identificado por infoHashHex, excluyendo peers eliminados.
func (t *Tracker) CountPeers(infoHashHex string) (complete, incomplete int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if sw := t.Torrents[infoHashHex]; sw != nil {
		for _, p := range sw.Peers {
			if p.Deleted {
				continue
			}
			if p.Completed {
				complete++
			} else {
				incomplete++
			}
		}
	}
	return
}

// Utility: hex encode 20 bytes (len=20) to hex string
// Bytes20ToHex valida que b tenga exactamente 20 bytes (p. ej. SHA-1) y
// devuelve su representación en hexadecimal para usar como clave/persistencia.
func Bytes20ToHex(b []byte) (string, error) {
	if len(b) != 20 {
		return "", errors.New("expected 20 bytes")
	}
	buf := make([]byte, hex.EncodedLen(20))
	hex.Encode(buf, b)
	return string(buf), nil
}

// StartSyncListener inicia el servidor de sincronización para recibir mensajes.
func (t *Tracker) StartSyncListener(listenAddr string) error {
	listener, err := NewSyncListener(t, listenAddr)
	if err != nil {
		return err
	}
	t.syncListener = listener
	t.syncListener.Start()
	return nil
}

// StartSyncManager inicia el cliente de sincronización periódica.
func (t *Tracker) StartSyncManager(syncInterval time.Duration) {
	t.syncManager = NewSyncManager(t, t.remotePeers, syncInterval)
	t.syncManager.Start()
}

// StopSync detiene los procesos de sincronización.
func (t *Tracker) StopSync() {
	if t.syncListener != nil {
		t.syncListener.Stop()
	}
	if t.syncManager != nil {
		t.syncManager.Stop()
	}
}

// Paths
// dataTempPath devuelve la ruta temporal usada durante guardados atómicos.
func (t *Tracker) dataTempPath() string { return t.DataPath + ".tmp" }

// Ensure data directory exists
// ensureDir crea el directorio contenedor de path si no existe aún.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}
