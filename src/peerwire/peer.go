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

	// remote bitfield (as advertised by the peer). Length should be ceil(NumPieces/8)
	remoteBF []byte

	// simple per-peer download state (phase 1)
	curPiece    int // -1 if none
	curOffset   int // next offset within curPiece to request
	downloading bool
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

// UpdateRemoteBitfield stores the remote peer bitfield snapshot
func (p *PeerConn) UpdateRemoteBitfield(b []byte) {
	if b == nil {
		p.remoteBF = nil
		return
	}
	if p.manager != nil && p.manager.Store() != nil {
		// Trim/expand to expected length
		exp := (p.manager.Store().NumPieces() + 7) / 8
		if len(b) != exp {
			// ignore invalid sizes
			return
		}
	}
	p.remoteBF = make([]byte, len(b))
	copy(p.remoteBF, b)
}

// RemoteHasPiece checks remote bitfield for a piece index
func (p *PeerConn) RemoteHasPiece(i int) bool {
	if i < 0 || p.remoteBF == nil {
		return false
	}
	byteIdx := i / 8
	bit := 7 - (i % 8)
	if byteIdx < 0 || byteIdx >= len(p.remoteBF) {
		return false
	}
	return (p.remoteBF[byteIdx] & (1 << uint(bit))) != 0
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
