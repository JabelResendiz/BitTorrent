package peerwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
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

// funcion para leer mensaje del peer con reintentos y timeouts progresivos
func (p *PeerConn) ReadMessage() (id byte, payload []byte, err error) {
	// Reintentos con timeouts progresivos: 1s, 2s, 4s
	timeouts := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for attempt := 0; attempt < len(timeouts); attempt++ {
		if attempt > 0 {
			fmt.Printf("Reintento %d/%d para leer mensaje del peer (timeout: %v)\n", attempt+1, len(timeouts), timeouts[attempt])
		}

		id, payload, err = p.readMessageWithTimeout(timeouts[attempt])
		if err == nil {
			return id, payload, nil
		}

		// Si es el último intento, devolver el error
		if attempt == len(timeouts)-1 {
			return 0, nil, fmt.Errorf("fallo después de %d intentos: %w", len(timeouts), err)
		}
	}

	return 0, nil, err
}

// readMessageWithTimeout lee un mensaje con un timeout específico
func (p *PeerConn) readMessageWithTimeout(timeout time.Duration) (id byte, payload []byte, err error) {
	// Establecer deadline para la lectura
	if err := p.Conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return 0, nil, err
	}
	// Resetear deadline después de leer
	defer p.Conn.SetReadDeadline(time.Time{})

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
