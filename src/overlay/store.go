package overlay

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

// ProviderMeta representa a un peer que anunció un infohash
type ProviderMeta struct {
	Addr     string `json:"addr"`
	PeerId   string `json:"peer_id"`
	Left     int64  `json:"left"`
	LastSeen int64  `json:"last_seen"`
}

// Store mantiene el mapeo infoHash -> providers
type Store struct {
	mu      sync.RWMutex
	records map[string]map[string]ProviderMeta // infoHash -> addr -> meta
	ttl     time.Duration
}

// NewStore crea un store con TTL para providers stale
func NewStore(ttl time.Duration) *Store {
	return &Store{
		records: make(map[string]map[string]ProviderMeta),
		ttl:     ttl,
	}
}

// Announce agrega o actualiza un provider para un infoHash
func (s *Store) Announce(infoHash string, p ProviderMeta) error {
	if infoHash == "" || p.Addr == "" {
		return errors.New("invalid announce")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.records[infoHash]
	if !ok {
		m = make(map[string]ProviderMeta)
		s.records[infoHash] = m
	}
	p.LastSeen = time.Now().Unix()
	m[p.Addr] = p
	return nil
}

// Merge merges providers from another store payload (used by gossip)
func (s *Store) Merge(infoHash string, providers []ProviderMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.records[infoHash]
	if !ok {
		m = make(map[string]ProviderMeta)
		s.records[infoHash] = m
	}
	for _, p := range providers {
		existing, ex := m[p.Addr]
		if !ex || p.LastSeen > existing.LastSeen {
			m[p.Addr] = p
		}
	}
}

// Lookup devuelve una lista de provider addresses ordenadas por LastSeen (más recientes primero)
func (s *Store) Lookup(infoHash string, limit int) []ProviderMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []ProviderMeta{}
	m, ok := s.records[infoHash]
	if !ok {
		return out
	}
	cutoff := time.Now().Add(-s.ttl).Unix()
	for _, p := range m {
		if p.LastSeen >= cutoff {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen > out[j].LastSeen })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ToJSON returns JSON representation of providers for an infoHash
func (s *Store) ToJSON(infoHash string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.records[infoHash]
	if !ok {
		return json.Marshal([]ProviderMeta{})
	}
	list := make([]ProviderMeta, 0, len(m))
	for _, p := range m {
		list = append(list, p)
	}
	return json.Marshal(list)
}
