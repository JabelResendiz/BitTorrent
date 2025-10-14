package peerwire

import "sync"

type Manager struct {
	mu    sync.RWMutex
	peers map[*PeerConn]struct{}
	store PieceStore
}

func NewManager(store PieceStore) *Manager {
	m := &Manager{peers: make(map[*PeerConn]struct{}), store: store}
	if store != nil {
		store.OnPieceComplete(func(idx int) { m.BroadcastHave(idx) })
	}
	return m
}

func (m *Manager) AddPeer(p *PeerConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[p] = struct{}{}
}

func (m *Manager) RemovePeer(p *PeerConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, p)
}

func (m *Manager) BroadcastHave(index int) {
	m.mu.RLock()
	peers := make([]*PeerConn, 0, len(m.peers))
	for p := range m.peers {
		peers = append(peers, p)
	}
	m.mu.RUnlock()
	// debug log
	// Note: keep it simple; avoid import cycles by not using log package here.
	// The client will see this on stdout.
	println("Broadcast HAVE a", len(peers), "peers. Pieza", index)
	for _, p := range peers {
		_ = p.SendHave(uint32(index))
	}
}

func (m *Manager) Store() PieceStore { return m.store }

// HasPeerAddr returns true if there is already a peer with the given remote address (ip:port)
func (m *Manager) HasPeerAddr(addr string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for p := range m.peers {
		if p != nil && p.Conn != nil && p.Conn.RemoteAddr() != nil && p.Conn.RemoteAddr().String() == addr {
			return true
		}
	}
	return false
}
