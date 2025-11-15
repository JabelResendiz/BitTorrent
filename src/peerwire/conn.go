package peerwire

import (
	"fmt"
	"net"
	"time"
)

func NewPeerConn(addr string, infoHash [20]byte, peerId [20]byte) (*PeerConn, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error conectando al peer %s: %v", addr, err)
	}

	return &PeerConn{
		Conn:           conn,
		InfoHash:       infoHash,
		PeerId:         peerId,
		AmChoking:      true,
		AmInterested:   false,
		PeerChoking:    true,
		PeerInterested: false,
	}, nil
}

// NewPeerConnFromConn envuelve una conexión existente (aceptada por un listener)
// para reutilizar la misma estructura y lógica de PeerConn.
func NewPeerConnFromConn(conn net.Conn, infoHash [20]byte, peerId [20]byte) *PeerConn {
	return &PeerConn{
		Conn:           conn,
		InfoHash:       infoHash,
		PeerId:         peerId,
		AmChoking:      true,
		AmInterested:   false,
		PeerChoking:    true,
		PeerInterested: false,
	}
}
