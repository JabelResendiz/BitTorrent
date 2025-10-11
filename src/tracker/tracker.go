package tracker

//tracker/tracker.go

import (
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Peer: estado mínimo de un peer en un swarm
type Peer struct {
	PeerIDHex string    `json:"peer_id"`
	IP        string    `json:"ip"` // IPv4 textual (ej.: 192.168.1.10)
	Port      uint16    `json:"port"`
	LastSeen  time.Time `json:"last_seen"`
	Completed bool      `json:"completed"` // true si left==0 reportado
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
}

// New crea una instancia de Tracker con configuración y estado iniciales.
// interval: segundos que los clientes deben esperar entre announces.
// timeout: tiempo para considerar inactivo a un peer (p. ej., 2×interval).
// maxPeers: máximo de peers a devolver por respuesta.
// dataPath: ruta al archivo JSON para persistencia.
func New(interval, timeout time.Duration, maxPeers int, dataPath string) *Tracker {
	return &Tracker{
		Torrents:     make(map[string]*Swarm),
		Interval:     interval,
		PeerTimeout:  timeout,
		MaxPeersResp: maxPeers,
		DataPath:     dataPath,
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
// Actualiza IP, puerto y LastSeen al momento now.
func (t *Tracker) AddPeer(infoHashHex, peerIDHex string, ip net.IP, port uint16, completed bool, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	sw := t.getOrCreateSwarm(infoHashHex)
	p := sw.Peers[peerIDHex]
	if p == nil {
		p = &Peer{PeerIDHex: peerIDHex}
		sw.Peers[peerIDHex] = p
	}
	p.IP = ip.String()
	p.Port = port
	p.LastSeen = now
	p.Completed = completed || p.Completed
}

// Remove peer
// RemovePeer elimina un peer por peerIDHex del swarm de infoHashHex.
// Si el swarm queda vacío, también elimina la entrada del torrent.
func (t *Tracker) RemovePeer(infoHashHex, peerIDHex string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	sw := t.Torrents[infoHashHex]
	if sw == nil {
		return
	}
	delete(sw.Peers, peerIDHex)
	if len(sw.Peers) == 0 {
		delete(t.Torrents, infoHashHex)
	}
}

// Get peers (excluding one) up to max
// GetPeers devuelve hasta max peers del swarm identificado por infoHashHex,
// excluyendo al peer con ID excludePeerIDHex. Devuelve copias para no exponer
// punteros internos.
func (t *Tracker) GetPeers(infoHashHex, excludePeerIDHex string, max int) []*Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	sw := t.Torrents[infoHashHex]
	if sw == nil {
		return nil
	}
	res := make([]*Peer, 0, max)
	for id, p := range sw.Peers {
		if id == excludePeerIDHex {
			continue
		}
		res = append(res, &Peer{PeerIDHex: p.PeerIDHex, IP: p.IP, Port: p.Port, LastSeen: p.LastSeen})
		if len(res) >= max {
			break
		}
	}
	return res
}

// Expire old peers
// GC recorre todos los swarms y elimina peers cuyo LastSeen sea mayor que
// PeerTimeout. También limpia swarms vacíos. Devuelve la cantidad de peers expirados.
func (t *Tracker) GC(now time.Time) (expired int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for ih, sw := range t.Torrents {
		for id, p := range sw.Peers {
			if now.Sub(p.LastSeen) > t.PeerTimeout {
				delete(sw.Peers, id)
				expired++
			}
		}
		if len(sw.Peers) == 0 {
			delete(t.Torrents, ih)
		}
	}
	return
}

// CountPeers retorna el número de seeders (complete) y leechers (incomplete)
// en el swarm identificado por infoHashHex.
func (t *Tracker) CountPeers(infoHashHex string) (complete, incomplete int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if sw := t.Torrents[infoHashHex]; sw != nil {
		for _, p := range sw.Peers {
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

// Paths
// dataTempPath devuelve la ruta temporal usada durante guardados atómicos.
func (t *Tracker) dataTempPath() string { return t.DataPath + ".tmp" }

// Ensure data directory exists
// ensureDir crea el directorio contenedor de path si no existe aún.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}
