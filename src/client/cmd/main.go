// client/cmd/main.go

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	// Flags: --torrent (obligatorio), --archives (opcional con default ./archives)
	torrentFlag := flag.String("torrent", "", "ruta al archivo .torrent (obligatorio)")
	archivesFlag := flag.String("archives", "./archives", "directorio donde guardar/leer archivos")
	flag.Parse()

	if *torrentFlag == "" {
		fmt.Println("Error: debe especificar --torrent=/ruta/al/archivo.torrent")
		os.Exit(2)
	}
	archivesDir := *archivesFlag
	// expand ~ if present
	if strings.HasPrefix(archivesDir, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if archivesDir == "~" {
				archivesDir = home
			} else if strings.HasPrefix(archivesDir, "~/") {
				archivesDir = filepath.Join(home, archivesDir[2:])
			}
		}
	}
	if err := os.MkdirAll(archivesDir, 0755); err != nil {
		fmt.Println("No se pudo crear directorio de archivos:", err)
		os.Exit(1)
	}

	// Abrir y decodificar el .torrent
	torrent, err := os.Open(*torrentFlag)
	if err != nil {
		panic(err)
	}
	defer torrent.Close()

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

	// Parse expected piece hashes (info.pieces) si está presente
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

	peerId := generatePeerId()

	// Abrir listener local (puerto asignado automáticamente)
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	listenPort := ln.Addr().(*net.TCPAddr).Port
	fmt.Println("Cliente escuchando en puerto:", listenPort)

	// Nombre de salida y rutas
	outName := "archivo.bin"
	if n, ok := info["name"].(string); ok && n != "" {
		outName = filepath.Base(n)
	}
	tempPath := filepath.Join(archivesDir, outName+".part")
	finalPath := filepath.Join(archivesDir, outName)

	// Elegir modo: si ya existe el archivo final con tamaño correcto, abrir en modo seeding
	useFinal := false
	if st, err := os.Stat(finalPath); err == nil && st.Size() == length {
		useFinal = true
	}
	var store *peerwire.DiskPieceStore
	if useFinal {
		store, err = peerwire.NewDiskPieceStoreWithMode(finalPath, int(pieceLength), length, false)
	} else {
		store, err = peerwire.NewDiskPieceStore(tempPath, int(pieceLength), length)
	}
	if err != nil {
		panic(err)
	}
	mgr := peerwire.NewManager(store)

	// Verificación SHA-1 por pieza si tenemos los hashes esperados
	if len(expectedHashes) == store.NumPieces() {
		store.SetExpectedHashes(expectedHashes)
	}

	// Si usamos archivo final existente, intentar marcar piezas completas (seeding)
	if useFinal && len(expectedHashes) == store.NumPieces() {
		if err := store.ScanAndMarkComplete(); err != nil {
			fmt.Println("No se pudo escanear archivo existente para seed:", err)
		}
	}

	// Renombrar .part -> final al completar todas las piezas (solo si estamos usando .part)
	store.OnPieceComplete(func(_ int) {
		n := store.NumPieces()
		all := true
		for i := 0; i < n; i++ {
			if !store.HasPiece(i) {
				all = false
				break
			}
		}
		if all && !useFinal {
			if err := os.Rename(tempPath, finalPath); err == nil {
				fmt.Println("Descarga completa. Archivo listo en:", finalPath)
			} else {
				fmt.Println("No se pudo renombrar el archivo final:", err)
			}
		}
	})

	// Anunciarse al tracker (started)
	// Calcular 'left' en base a piezas ya presentes (soporta seeding inmediato)
	computeLeft := func() int64 {
		var have int64
		num := store.NumPieces()
		for i := 0; i < num; i++ {
			if store.HasPiece(i) {
				if i == num-1 {
					have += length - int64(store.PieceLength())*int64(num-1)
				} else {
					have += int64(store.PieceLength())
				}
			}
		}
		if have > length {
			have = length
		}
		return length - have
	}
	params := url.Values{
		"peer_id":    []string{peerId},
		"port":       []string{fmt.Sprintf("%d", listenPort)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{fmt.Sprintf("%d", computeLeft())},
		"compact":    []string{"1"},
		"event":      []string{"started"},
		"numwant":    []string{"50"},
		"key":        []string{"jc12345"},
	}
	fullURL := announce + "?info_hash=" + infoHashEncoded + "&" + params.Encode()
	fmt.Println("Tracker request:", fullURL)
	resp, err := http.Get(fullURL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	trackerResponse, err := bencode.Decode(resp.Body)
	if err != nil && err != io.EOF {
		panic(err)
	}
	fmt.Println("Tracker responde:", trackerResponse)

	// Conectar a peers del tracker
	if peersRaw, ok := trackerResponse["peers"].(string); ok {
		data := []byte(peersRaw)
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
			pc.BindManager(mgr)
			if err := pc.Handshake(); err != nil {
				fmt.Println("Handshake fallido:", err)
				pc.Close()
				continue
			}
			fmt.Println("Conectado al peer, handshake OK")
			_ = pc.SendBitfield(store.Bitfield())
			pc.SendMessage(peerwire.MsgInterested, nil)
			go pc.ReadLoop()
		}
	}

	// Aceptar conexiones entrantes
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				fmt.Println("Error aceptando conexión:", err)
				continue
			}
			go func(conn net.Conn) {
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
					conn.Close()
					return
				}
				pc.BindManager(mgr)
				_ = pc.SendBitfield(store.Bitfield())
				go pc.ReadLoop()
			}(c)
		}
	}()

	// Enviar announce stopped al salir
	defer func() {
		left := length
		if store != nil {
			var have int64
			num := store.NumPieces()
			for i := 0; i < num; i++ {
				if store.HasPiece(i) {
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

	// Mantener el proceso vivo
	select {}
}
