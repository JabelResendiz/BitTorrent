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

// funcion para leer mensaje del peer
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

// SendBlockRequest sends a REQUEST message for a specific block
func (p *PeerConn) SendBlockRequest(index uint32, begin uint32, length uint32) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], index)
	binary.BigEndian.PutUint32(payload[4:8], begin)
	binary.BigEndian.PutUint32(payload[8:12], length)
	return p.SendMessage(MsgRequest, payload)
}

// SendPiece sends a piece block with given index, begin and data
func (p *PeerConn) SendPiece(index uint32, begin uint32, data []byte) error {
	hdr := new(bytes.Buffer)
	total := uint32(9 + len(data))
	if err := binary.Write(hdr, binary.BigEndian, total); err != nil {
		return err
	}
	hdr.WriteByte(MsgPiece)
	if err := binary.Write(hdr, binary.BigEndian, index); err != nil {
		return err
	}
	if err := binary.Write(hdr, binary.BigEndian, begin); err != nil {
		return err
	}
	if _, err := p.Conn.Write(append(hdr.Bytes(), data...)); err != nil {
		return err
	}
	return nil
}
