

package peerwire

import (
	"net"
	"fmt"
	"time"
)


func NewPeerConn(addr string, infoHash [20]byte, peerId [20]byte) (*PeerConn , error) {
    conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
    if err != nil {
        return nil, fmt.Errorf("error conectando al peer %s: %v", addr, err)
    }

    return &PeerConn{
        Conn: conn,
        InfoHash: infoHash,
        PeerId: peerId,
        AmChoking: true,
        AmInterested: false,
        PeerChoking: true,
        PeerInterested: false,
    }, nil
}


