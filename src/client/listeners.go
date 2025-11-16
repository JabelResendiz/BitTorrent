package client

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"src/peerwire"
)

func StartListeningForIncomingPeers(ln net.Listener, infoHash [20]byte, peerId string,
	store *peerwire.DiskPieceStore, mgr *peerwire.Manager) {

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				fmt.Println("Error aceptando conexión:", err)
				continue
			}
			go handleIncomingPeerConnection(c, infoHash, peerId, store, mgr)
		}
	}()
}

func handleIncomingPeerConnection(conn net.Conn, infoHash [20]byte, peerId string,
	store *peerwire.DiskPieceStore, mgr *peerwire.Manager) {

	//defer conn.Close()

	hs := make([]byte, peerwire.HandshakeLen)
	if _, err := io.ReadFull(conn, hs); err != nil {
		fmt.Println("Error leyendo handshake entrante:", err)
		conn.Close()
		return
	}

	if int(hs[0]) != 19 || string(hs[1:20]) != "BitTorrent protocol" {
		fmt.Println("Handshake entrante inválido: pstr")
		conn.Close()
		return
	}

	if !bytes.Equal(hs[28:48], infoHash[:]) {
		fmt.Println("Handshake entrante inválido: info_hash")
		conn.Close()
		return
	}

	var pidBytes [20]byte
	copy(pidBytes[:], []byte(peerId))
	pc := peerwire.NewPeerConnFromConn(conn, infoHash, pidBytes)

	if err := pc.SendHandshakeOnly(); err != nil {
		fmt.Println("Error enviando handshake de respuesta:", err)
		return
	}

	pc.BindManager(mgr)
	_ = pc.SendBitfield(store.Bitfield())

	go pc.ReadLoop()

}
