

package peerwire

import (
	"fmt"
	"encoding/binary"
)
func(p* PeerConn) ReadLoop() {
	for {
		id, payload, err := p.ReadMessage()
		if err != nil {
			fmt.Println("Error con peer:", err)
			p.Close()
			return 
		}


		p.handleMessage(id,payload)
	}
}

// actualiza estado segun el tipo de mensaje
func (p* PeerConn) handleMessage(id byte, payload[]byte) {
	switch id {
	case MsgChoke:
		p.PeerChoking = true
	case MsgUnchoke:
		p.PeerChoking = false
	case MsgInterested:
		p.PeerInterested = true
	case MsgNotInterested:
		p.PeerInterested= false
	case MsgHave:
		index := binary.BigEndian.Uint32(payload)
		fmt.Println("Peer tiene pieza:", index)
	case MsgBitfiled:
		fmt.Println("Bitfield inicial recibido")
	case MsgPort:
		fmt.Println("Puerto abierto")
	case 255:
		//ignorar
	default:
		fmt.Println("Mensaje desconocido",id)
	}
}