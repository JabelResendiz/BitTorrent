

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


// estructura para representar una pieza del archivo torrent
type Piece struct {
	Index int // posicion de la pieza dentro del torrent
	Data []byte // bytes descargados (puede estar incompleta)
	Hash [20]byte // hash SHA1 esperado de esta pieza 
	Complete bool // indica si ya se descargó y verificó
}

// representa el estado global del torrent
type TorrentState struct {
	Piece []Piece // slice de piezas 
	Bitfield []bool // mapa de que piezas se tienen 
}