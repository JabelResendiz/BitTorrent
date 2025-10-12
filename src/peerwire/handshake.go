

package peerwire

import (
	"bytes"
	"fmt"
	"io"
)

const (
	pstr = "BitTorrent protocol"
	pstrlen = 19
	HandshakeLen = 49 + pstrlen
)


// funcion donde se envia el handshake inicial y valida el recibido
func (p *PeerConn) Handshake() error {
	buf := new(bytes.Buffer)

	buf.WriteByte(pstrlen)
	buf.WriteString(pstr)
	buf.Write(make([]byte,8)) // reservar 8 bytes
	buf.Write(p.InfoHash[:])
	buf.Write(p.PeerId[:])

	//enviarlo
	if _,err := p.Conn.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("error enviando handshake: %v", err)
	}

	// leer respuesta
	resp := make([]byte, HandshakeLen)
	if _, err := io.ReadFull(p.Conn,resp) ; err != nil {
		return fmt.Errorf("error leyendo el handshake: %v",err)
	}

	if int(resp[0]) != pstrlen || string(resp[1:20]) != pstr {
		return fmt.Errorf("handshake invalido: pstr incorrecto")
	}

	// validar el infohash
	if !bytes.Equal(resp[28:48], p.InfoHash[:]) {
		return fmt.Errorf("info_hash no coincide")
	}
	
	fmt.Println("Handshake completado con exito")
	return nil
} 