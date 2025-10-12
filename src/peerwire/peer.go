package peerwire

import "net"

type PeerConn struct {
	Conn           net.Conn
	InfoHash       [20]byte
	PeerId         [20]byte
	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool
	manager        *Manager
}

func (p *PeerConn) Close() {
	if p.manager != nil {
		p.manager.RemovePeer(p)
	}
	p.Conn.Close()
}

// BindManager attaches this peer connection to a Manager for broadcasting
func (p *PeerConn) BindManager(m *Manager) {
	p.manager = m
	if m != nil {
		m.AddPeer(p)
	}
}

// estructura para representar una pieza del archivo torrent
type Piece struct {
	Index    int      // posicion de la pieza dentro del torrent
	Data     []byte   // bytes descargados (puede estar incompleta)
	Hash     [20]byte // hash SHA1 esperado de esta pieza
	Complete bool     // indica si ya se descargó y verificó
}

// representa el estado global del torrent
type TorrentState struct {
	Piece    []Piece // slice de piezas
	Bitfield []bool  // mapa de que piezas se tienen
}
