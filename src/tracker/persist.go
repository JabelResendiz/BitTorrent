package tracker

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type trackerDisk struct {
	Torrents map[string]*Swarm `json:"torrents"`
}

// LoadFromFile loads tracker data from JSON file if it exists.
// Carga el estado del tracker desde el archivo JSON configurado (DataPath).
// Si el archivo no existe, deja el estado vacío. Filtra peers demasiado
// antiguos para evitar reintroducir estado obsoleto.
func (t *Tracker) LoadFromFile() error {
	if t.DataPath == "" {
		return nil
	}
	f, err := os.Open(t.DataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var disk trackerDisk
	if err := dec.Decode(&disk); err != nil {
		return err
	}
	// Basic sanity: ensure maps
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Torrents == nil {
		t.Torrents = make(map[string]*Swarm)
	}
	for ih, sw := range disk.Torrents {
		if sw == nil || sw.Peers == nil {
			continue
		}
		// Filter out peers with ancient LastSeen beyond timeout (start fresh)
		clean := &Swarm{Peers: make(map[string]*Peer)}
		for pid, p := range sw.Peers {
			if p == nil {
				continue
			}
			if time.Since(p.LastSeen) > t.PeerTimeout*4 { // drop very old
				continue
			}
			clean.Peers[pid] = p
		}
		if len(clean.Peers) > 0 {
			t.Torrents[ih] = clean
		}
	}
	return nil
}

// SaveToFile saves tracker data atomically to JSON file.
// Guarda el estado actual del tracker en el archivo JSON configurado (DataPath)
// de forma atómica (escribiendo primero a un archivo temporal y luego renombrando).
// Toma un snapshot bajo RLock para evitar bloquear a los handlers durante la
// serialización.
func (t *Tracker) SaveToFile() error {
	if t.DataPath == "" {
		return nil
	}
	if err := ensureDir(t.DataPath); err != nil {
		return err
	}
	// Create snapshot under read lock
	t.mu.RLock()
	snapshot := trackerDisk{Torrents: make(map[string]*Swarm, len(t.Torrents))}
	for ih, sw := range t.Torrents {
		copySw := &Swarm{Peers: make(map[string]*Peer, len(sw.Peers))}
		for pid, p := range sw.Peers {
			copyP := *p
			copySw.Peers[pid] = &copyP
		}
		snapshot.Torrents[ih] = copySw
	}
	t.mu.RUnlock()

	// Write to temp file then rename
	tmp := t.dataTempPath()
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&snapshot); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, t.DataPath); err != nil {
		return err
	}
	return nil
}

// SaveOnChange is a helper to wrap an operation and save afterwards.
// Ejecuta la operación de actualización de estado op() y luego persiste el
// estado llamando a SaveToFile. Devuelve error si la persistencia falla.
func (t *Tracker) SaveOnChange(op func()) error {
	op()
	if err := t.SaveToFile(); err != nil {
		return fmt.Errorf("save failed: %w", err)
	}
	return nil
}

// DrainTo discards the remaining content from r (best-effort); utility for handlers.
// Drena y cierra el cuerpo HTTP para permitir reutilización de conexiones y
// evitar fugas. Es una utilidad opcional para handlers.
func DrainTo(r io.ReadCloser) {
	if r != nil {
		io.Copy(io.Discard, r)
		r.Close()
	}
}
