package peerwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func (p *PeerConn) ReadLoop() {
	for {
		id, payload, err := p.ReadMessage()
		if err != nil {
			fmt.Println("Error con peer:", err)
			p.Close()
			return
		}

		p.handleMessage(id, payload)
	}
}

// actualiza estado segun el tipo de mensaje
func (p *PeerConn) handleMessage(id byte, payload []byte) {
	switch id {
	case MsgChoke:
		p.PeerChoking = true
	case MsgUnchoke:
		p.PeerChoking = false
		fmt.Println("Peer te unchokeo. Enviando request de prueba...")

		// Enviar un request de bloque real (offset 0, length acorde al tamaño de la pieza)
		index := uint32(0)
		begin := uint32(0)
		length := uint32(16384)
		if p.manager != nil && p.manager.Store() != nil {
			pl := p.manager.Store().PieceLength()
			tl := p.manager.Store().TotalLength()
			// pieza 0 puede ser más corta si total < pieceLength
			req := int64(pl)
			if tl < int64(pl) {
				req = tl
			}
			if req <= 0 {
				req = int64(pl)
			}
			if req > int64(^uint32(0)) {
				req = int64(^uint32(0))
			}
			length = uint32(req)
		}
		req := new(bytes.Buffer)
		binary.Write(req, binary.BigEndian, index)
		binary.Write(req, binary.BigEndian, begin)
		binary.Write(req, binary.BigEndian, length)

		if err := p.SendMessage(MsgRequest, req.Bytes()); err != nil {
			fmt.Println("Error enviando request:", err)
		}

	case MsgInterested:
		p.PeerInterested = true
	case MsgNotInterested:
		p.PeerInterested = false
	case MsgHave:
		index := binary.BigEndian.Uint32(payload)
		fmt.Println("Peer tiene pieza:", index)
	case MsgBitfiled:
		fmt.Println("Bitfield inicial recibido")
	case MsgPort:
		fmt.Println("Puerto abierto")
	case MsgPiece:
		if len(payload) < 8 {
			fmt.Println("Piece demasiado corta")
			return
		}
		index := binary.BigEndian.Uint32(payload[0:4])
		begin := binary.BigEndian.Uint32(payload[4:8])
		block := payload[8:]
		fmt.Printf("Recibido block de pieza %d, offset %d, tamaño %d bytes\n", index, begin, len(block))
		// if attached to a manager with a store, persist block
		if p.manager != nil && p.manager.Store() != nil {
			if _, err := p.manager.Store().WriteBlock(int(index), int(begin), block); err != nil {
				fmt.Println("Error guardando bloque:", err)
			}
		}
	case 255:
		//ignorar
	default:
		fmt.Println("Mensaje desconocido", id)
	}
}
