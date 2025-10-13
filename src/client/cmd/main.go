// client/cmd/main.go

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"src/bencode"
	"src/peerwire"
	"strings"
)

func generatePeerId() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("-JC0001-%s", hex.EncodeToString(buf))
}

func main() {

	torrent, err := os.Open("test.torrent")

	if err != nil {
		panic(err)
	}

	meta, err := bencode.Decode(torrent)

	if err != nil {
		panic(err)
	}

	announce := meta["announce"].(string)
	info := meta["info"].(map[string]interface{})

	infoEncoded := bencode.Encode(info)
	infoHash := sha1.Sum(infoEncoded)

	var length int64
	if v, ok := info["length"].(int64); ok {
		length = v
	}
	var pieceLength int64
	if v, ok := info["piece length"].(int64); ok {
		pieceLength = v
	}

	// Parse expected piece hashes (info.pieces) if present and well-formed
	var expectedHashes [][20]byte
	if piecesRaw, ok := info["pieces"].(string); ok {
		numPieces := int((length + pieceLength - 1) / pieceLength)
		if len(piecesRaw) == numPieces*20 {
			expectedHashes = make([][20]byte, numPieces)
			for i := 0; i < numPieces; i++ {
				copy(expectedHashes[i][:], piecesRaw[i*20:(i+1)*20])
			}
		} else {
			fmt.Printf("Advertencia: longitud de 'pieces' (%d) no coincide con numPieces*20 (%d)\n", len(piecesRaw), numPieces*20)
		}
	}

	var buf strings.Builder
	for _, b := range infoHash {
		buf.WriteString(fmt.Sprintf("%%%02X", b))
	}

	infoHashEncoded := buf.String()

	//peerId := "-JC0001-123456789012"
	peerId := generatePeerId()

	// hostIP := os.Getenv("MY_PEER_IP")
	// if hostIP == "" {
	// 	hostIP = "127.0.0.1"
	// }

	// Abrir listener local para aceptar conexiones entrantes; usar puerto 0 para que el SO asigne uno libre
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	listenPort := ln.Addr().(*net.TCPAddr).Port
	fmt.Println("Cliente escuchando en puerto:", listenPort)

	params := url.Values{
		"peer_id":    []string{peerId},
		"port":       []string{fmt.Sprintf("%d", listenPort)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{fmt.Sprintf("%d", length)},
		"compact":    []string{"1"},
		"event":      []string{"started"},
		"numwant":    []string{"50"},
		"key":        []string{"jc12345"},
		//"ip":         []string{hostIP},
	}

	fullURL := announce + "?info_hash=" + infoHashEncoded + "&" + params.Encode()
	fmt.Println("Tracker request: ", fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	// decodificar la respuesta del tracker

	trackerResponse, err := bencode.Decode(resp.Body)
	if err != nil && err != io.EOF {
		panic(err)
	}

	fmt.Println("Tracker responde: ", trackerResponse)

	// Inicializar almacenamiento en disco y manager para broadcast HAVE
	store, err := peerwire.NewDiskPieceStore("download.data", int(pieceLength), length)
	if err != nil {
		panic(err)
	}
	mgr := peerwire.NewManager(store)

	// Si tenemos hashes esperados válidos, configurarlos para verificación SHA-1
	if len(expectedHashes) == store.NumPieces() {
		store.SetExpectedHashes(expectedHashes)
	}

	// Enviar announce stopped al salir para limpiar del tracker
	defer func() {
		// calcular left según piezas completas en store
		left := length
		if store != nil {
			var have int64
			num := store.NumPieces()
			for i := 0; i < num; i++ {
				if store.HasPiece(i) {
					// suma tamaño real de la pieza i
					if i == num-1 {
						have += length - int64(store.PieceLength())*int64(num-1)
					} else {
						have += int64(store.PieceLength())
					}
				}
			}
			if have > left {
				have = left
			}
			left = length - have
		}
		stopParams := url.Values{
			"peer_id":    []string{peerId},
			"port":       []string{fmt.Sprintf("%d", listenPort)},
			"uploaded":   []string{"0"},
			"downloaded": []string{fmt.Sprintf("%d", length-left)},
			"left":       []string{fmt.Sprintf("%d", left)},
			"compact":    []string{"1"},
			"event":      []string{"stopped"},
			"numwant":    []string{"0"},
			"key":        []string{"jc12345"},
		}
		stopURL := announce + "?info_hash=" + infoHashEncoded + "&" + stopParams.Encode()
		_, _ = http.Get(stopURL)
	}()

	if peersRaw, ok := trackerResponse["peers"].(string); ok {
		data := []byte(peersRaw)

		// deduplicar ip:port
		seen := make(map[string]struct{})

		for i := 0; i < len(data); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
			port := binary.BigEndian.Uint16(data[i+4 : i+6])
			addr := fmt.Sprintf("%s:%d", ip, port)
			if _, dup := seen[addr]; dup {
				fmt.Printf("Peer duplicado omitido: %s\n", addr)
				continue
			}
			seen[addr] = struct{}{}
			fmt.Printf("Peer: %s\n", addr)

			var peerIdBytes [20]byte
			copy(peerIdBytes[:], []byte(peerId))
			pc, err := peerwire.NewPeerConn(addr, infoHash, peerIdBytes)

			if err != nil {
				fmt.Println("Error creando PeerConn:", err)
				continue
			}

			defer pc.Close()
			// Integrar con el manager para almacenamiento y broadcast HAVE
			pc.BindManager(mgr)

			// Handshake
			if err := pc.Handshake(); err != nil {
				panic(err)
			}

			fmt.Println("Conectado al peer, handshake OK")

			// Enviar nuestro bitfield inicial (puede estar vacío al inicio)
			_ = pc.SendBitfield(store.Bitfield())

			//enviar el Interested (también será enviado automáticamente si procede al recibir bitfield remoto)
			pc.SendMessage(peerwire.MsgInterested, nil)

			// iniciar loop de lectura en paralelo
			go pc.ReadLoop()
		}
	}

	// Aceptar conexiones entrantes en segundo plano
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				fmt.Println("Error aceptando conexión:", err)
				continue
			}
			go func(conn net.Conn) {
				// Leer handshake del peer remoto
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
				// Enviar nuestro handshake de respuesta
				var pidBytes [20]byte
				copy(pidBytes[:], []byte(peerId))
				pc := peerwire.NewPeerConnFromConn(conn, infoHash, pidBytes)
				if err := pc.SendHandshakeOnly(); err != nil {
					fmt.Println("Error enviando handshake de respuesta:", err)
					conn.Close()
					return
				}
				// Integrar con manager, enviar Bitfield y arrancar ReadLoop
				pc.BindManager(mgr)
				_ = pc.SendBitfield(store.Bitfield())
				go pc.ReadLoop()
			}(c)
		}
	}()

	// mantener main corriendo mientras llegan mensajes
	select {}
}
