// client/cmd/main.go

package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
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

	params := url.Values{
		"peer_id":    []string{peerId},
		"port":       []string{"6881"},
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

	if peersRaw, ok := trackerResponse["peers"].(string); ok {
		data := []byte(peersRaw)

		for i := 0; i < len(data); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
			port := binary.BigEndian.Uint16(data[i+4 : i+6])
			fmt.Printf("Peer: %s:%d\n", ip, port)

			addr := fmt.Sprintf("%s:%d", ip, port)
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

			//enviar el Interested
			pc.SendMessage(peerwire.MsgInterested, nil)

			// iniciar loop de lectura en paralelo
			go pc.ReadLoop()
		}
	}

	// mantener main corriendo meintras llegan mensajes
	select {}
}
