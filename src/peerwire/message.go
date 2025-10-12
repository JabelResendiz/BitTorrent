package peerwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	MsgChoke         = 0
	MsgUnchoke       = 1
	MsgInterested    = 2
	MsgNotInterested = 3
	MsgHave          = 4
	MsgBitfiled      = 5
	MsgRequest       = 6
	MsgPiece         = 7
	MsgCancel        = 8
	MsgPort          = 9
)

// funcion que envia un mensaje generico del protocolo
func (p *PeerConn) SendMessage(id byte, payload []byte) error {
	length := uint32(1 + len(payload))
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, length); err != nil {
		return err
	}
	buf.WriteByte(id)
	if payload != nil {
		buf.Write(payload)
	}

	_, err := p.Conn.Write(buf.Bytes())
	return err
}

// func WriteMessage(w io.Writer, id byte, payload []byte) error {
// 	length := uint32(1 + len(payload))
// 	if id == 255 { // keep-alive indicator, we'll use id=255 to mean none
// 	  length = 0
// 	  return binary.Write(w, binary.BigEndian, length) // keep-alive
// 	}
// 	if err := binary.Write(w, binary.BigEndian, length); err != nil {
// 	  return err
// 	}
// 	if err := binary.Write(w, binary.BigEndian, id); err != nil {
// 	  return err
// 	}
// 	if len(payload) > 0 {
// 	  _, err := w.Write(payload)
// 	  return err
// 	}
// 	return nil
//   }

// funcion para leer mesnaje del peer
func (p *PeerConn) ReadMessage() (id byte, payload []byte, err error) {
	var length uint32
	if err := binary.Read(p.Conn, binary.BigEndian, &length); err != nil {
		return 0, nil, err
	}

	if length == 0 { //keep-alive
		return 255, nil, nil
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(p.Conn, data); err != nil {
		return 0, nil, err
	}

	return data[0], data[1:], nil
}

// SendHave sends a HAVE message for the given piece index
func (p *PeerConn) SendHave(index uint32) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, index)
	fmt.Println("Enviando HAVE de pieza", index)
	return p.SendMessage(MsgHave, payload)
}

// SendBitfield sends the current bitfield (if non-empty)
func (p *PeerConn) SendBitfield(bits []byte) error {
	if len(bits) == 0 {
		return nil
	}
	// don't send if all zeros (no piezas): es válido omitir el bitfield
	allZero := true
	for _, b := range bits {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return nil
	}
	return p.SendMessage(MsgBitfiled, bits)
}
