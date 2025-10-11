

package peerwire

import "net"

type PeerConn struct {
	Conn net.Conn
	InfoHash [20]byte
	PeerId   [20]byte
	AmChoking bool
	AmInterested bool
	PeerChoking bool
	PeerInterested bool
}


func (p* PeerConn) Close() {
	p.Conn.Close()
}