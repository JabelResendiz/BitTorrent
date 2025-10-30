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
	"os/signal"
	"path/filepath"
	"src/bencode"
	"src/peerwire"
	"strings"
	"sync"
	"syscall"
	"time"
)

func generatePeerId() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("-JC0001-%s", hex.EncodeToString(buf))
}

// sendAnnounce envía un announce al tracker y devuelve la respuesta decodificada
func sendAnnounce(announceURL, infoHashEncoded, peerId string, port int, uploaded, downloaded, left int64, event string, hostname string) (map[string]interface{}, error) {
	params := url.Values{
		"peer_id":    []string{peerId},
		"port":       []string{fmt.Sprintf("%d", port)},
		"uploaded":   []string{fmt.Sprintf("%d", uploaded)},
		"downloaded": []string{fmt.Sprintf("%d", downloaded)},
		"left":       []string{fmt.Sprintf("%d", left)},
		"compact":    []string{"1"},
		"key":        []string{"jc12345"},
		"hostname":   []string{hostname},
	}

	// Agregar IP externa si se proporciona (para Docker/NAT)
	// if externalIP != "" {
	// 	params.Set("ip", externalIP)
	// 	fmt.Printf("[ANNOUNCE] Usando IP externa: %s\n", externalIP)
	// }

	// Solo agregar event si no está vacío
	if event != "" {
		params.Set("event", event)
	}

	// Agregar numwant para started
	if event == "started" {
		params.Set("numwant", "50")
	} else if event == "stopped" {
		params.Set("numwant", "0")
	}

	fullURL := announceURL + "?info_hash=" + infoHashEncoded + "&" + params.Encode()

	if event != "" {
		fmt.Printf("[ANNOUNCE] Enviando event=%s, left=%d\n", event, left)
	} else {
		fmt.Printf("[ANNOUNCE] Enviando announce periódico, left=%d\n", left)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error en request: %w", err)
	}
	defer resp.Body.Close()

	// Decodificar respuesta
	trackerResponse, err := bencode.Decode(resp.Body)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error decodificando respuesta: %w", err)
	}

	// Verificar failure reason
	if failureReason, ok := trackerResponse["failure reason"].(string); ok {
		return trackerResponse, fmt.Errorf("tracker error: %s", failureReason)
	}

	return trackerResponse, nil
}

// sendScrape envía una petición scrape al tracker y muestra las estadísticas
func sendScrape(announceURL, infoHashEncoded string, infoHash [20]byte) {
	// Construir URL de scrape
	pos := strings.LastIndex(announceURL, "/")
	if pos == -1 {
		fmt.Println("[SCRAPE] URL inválida, no se puede hacer scrape")
		return
	}

	last := announceURL[pos+1:]
	if !strings.HasPrefix(last, "announce") {
		fmt.Println("[SCRAPE] Tracker no soporta scrape")
		return
	}

	scrapeURL := announceURL[:pos+1] + strings.Replace(last, "announce", "scrape", 1)
	fullURL := scrapeURL + "?info_hash=" + infoHashEncoded

	fmt.Println("[SCRAPE] Obteniendo estadísticas del tracker...")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		fmt.Println("[SCRAPE] Error:", err)
		return
	}
	defer resp.Body.Close()

	scrapeResponse, err := bencode.Decode(resp.Body)
	if err != nil && err != io.EOF {
		fmt.Println("[SCRAPE] Error decodificando:", err)
		return
	}

	// Extraer y mostrar estadísticas
	files, ok := scrapeResponse["files"].(map[string]interface{})
	if !ok {
		fmt.Println("[SCRAPE] No hay estadísticas disponibles")
		return
	}

	stats, ok := files[string(infoHash[:])].(map[string]interface{})
	if !ok {
		fmt.Println("[SCRAPE] No hay estadísticas para este torrent")
		return
	}

	complete, _ := stats["complete"].(int64)
	incomplete, _ := stats["incomplete"].(int64)
	downloaded, _ := stats["downloaded"].(int64)

	fmt.Println("\n========================================")
	fmt.Println("      ESTADÍSTICAS DEL TRACKER           ")
	fmt.Println("=========================================")
	fmt.Printf(" Seeders (completos):   %15d \n", complete)
	fmt.Printf(" Leechers (descargando): %14d \n", incomplete)
	fmt.Printf(" Descargas completadas:  %14d \n", downloaded)
	fmt.Printf(" Total peers:            %14d \n", complete+incomplete)
	fmt.Println("=========================================")
	fmt.Println()
}

func main() {
	// Variables de control para manejo de eventos
	var (
		sigChan           = make(chan os.Signal, 1)
		shutdownChan      = make(chan struct{})
		completedChan     = make(chan struct{})
		downloadCompleted bool
		completedMu       sync.Mutex
	)

	// Flags: --torrent (obligatorio), --archives (opcional con default ./archives)
	torrentFlag := flag.String("torrent", "", "ruta al archivo .torrent (obligatorio)")
	archivesFlag := flag.String("archives", "./archives", "directorio donde guardar/leer archivos")
	hostnameFlag := flag.String("hostname", "", "nombre de host para announces (requerido en Docker/NAT)")
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

	// ln, err := net.Listen("tcp",fmt.Sprintf(":%d",*port))
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

	// Elegir modo de apertura del store:
	// - si existe archivo final completo: seeding
	// - si existe .part del tamaño correcto: reanudar sin truncar
	// - si no existe: crear .part nuevo truncando
	useFinal := false
	usePartResume := false
	if st, err := os.Stat(finalPath); err == nil && st.Size() == length {
		useFinal = true
	} else if st, err := os.Stat(tempPath); err == nil && st.Size() == length {
		usePartResume = true
	}
	var store *peerwire.DiskPieceStore
	if useFinal {
		store, err = peerwire.NewDiskPieceStoreWithMode(finalPath, int(pieceLength), length, false)
	} else if usePartResume {
		store, err = peerwire.NewDiskPieceStoreWithMode(tempPath, int(pieceLength), length, false)
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

	// Si existe archivo final o .part previo, intentar marcar piezas completas por SHA-1
	if (useFinal || usePartResume) && len(expectedHashes) == store.NumPieces() {
		if err := store.ScanAndMarkComplete(); err != nil {
			fmt.Println("No se pudo escanear archivo existente para marcar piezas:", err)
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

				// Notificar que la descarga se completó
				completedMu.Lock()
				if !downloadCompleted {
					downloadCompleted = true
					close(completedChan)
				}
				completedMu.Unlock()
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

	// Enviar announce inicial con event=started
	initialLeft := computeLeft()
	trackerResponse, err := sendAnnounce(announce, infoHashEncoded, peerId, listenPort, 0, 0, initialLeft, "started", *hostnameFlag)
	if err != nil {
		panic(fmt.Errorf("error en announce inicial: %w", err))
	}
	fmt.Println("Tracker responde:", trackerResponse)

	// Hacer scrape para obtener estadísticas del torrent
	sendScrape(announce, infoHashEncoded, infoHash)

	// Extraer intervalo del tracker (por defecto 30 minutos)
	trackerInterval := 1800 * time.Second
	if intervalRaw, ok := trackerResponse["interval"].(int64); ok {
		trackerInterval = time.Duration(intervalRaw) * time.Second
		fmt.Printf("Intervalo de announces: %v\n", trackerInterval)
	}

	// Conectar a peers del tracker
	// Soportar tanto formato compact (string de bytes) como non-compact (lista de diccionarios)
	var peerAddrs []string

	if peersRaw, ok := trackerResponse["peers"].(string); ok {
		// Formato compact: 6 bytes por peer (4 IP + 2 puerto)
		data := []byte(peersRaw)
		for i := 0; i < len(data); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
			port := binary.BigEndian.Uint16(data[i+4 : i+6])
			addr := fmt.Sprintf("%s:%d", ip, port)
			peerAddrs = append(peerAddrs, addr)
		}
	} else if peersList, ok := trackerResponse["peers"].([]interface{}); ok {
		// Formato non-compact: lista de diccionarios {"ip": "hostname", "port": 12345}
		for _, peerRaw := range peersList {
			if peerDict, ok := peerRaw.(map[string]interface{}); ok {
				var ip string
				var port int64

				if ipVal, ok := peerDict["ip"].(string); ok {
					ip = ipVal
				}
				if portVal, ok := peerDict["port"].(int64); ok {
					port = portVal
				}

				if ip != "" && port > 0 {
					addr := fmt.Sprintf("%s:%d", ip, port)
					peerAddrs = append(peerAddrs, addr)
				}
			}
		}
	}

	// Conectar a los peers encontrados
	seen := make(map[string]struct{})
	for _, addr := range peerAddrs {
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

	// Goroutine: Announces periódicos al tracker
	go func() {
		ticker := time.NewTicker(trackerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				left := computeLeft()
				_, err := sendAnnounce(announce, infoHashEncoded, peerId, listenPort, 0, 0, left, "", *hostnameFlag)
				if err != nil {
					fmt.Println("[ERROR] Announce periódico fallido:", err)
				}

			case <-shutdownChan:
				return
			}
		}
	}()

	// Goroutine: Detectar completación y enviar event=completed
	go func() {
		<-completedChan
		fmt.Println("[INFO] Enviando event=completed al tracker...")
		_, err := sendAnnounce(announce, infoHashEncoded, peerId, listenPort, 0, length, 0, "completed", *hostnameFlag)
		if err != nil {
			fmt.Println("[ERROR] No se pudo enviar completed:", err)
		} else {
			fmt.Println("[INFO] Ahora soy un seeder completo")
		}
	}()

	// Configurar captura de señales del sistema
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	// Loop principal: esperar señal de terminación
	fmt.Println("\n=== Cliente BitTorrent ejecutándose ===")
	fmt.Println("Presiona Ctrl+C para detener el cliente")
	fmt.Printf("Escuchando en puerto: %d\n", listenPort)
	fmt.Printf("Announces cada: %v\n\n", trackerInterval)

	sig := <-sigChan

	// Shutdown limpio
	fmt.Printf("\n\n=== Señal %v recibida, iniciando shutdown limpio ===\n", sig)

	// Notificar a todas las goroutines que deben detenerse
	close(shutdownChan)

	// Enviar stopped al tracker
	fmt.Println("[SHUTDOWN] Enviando event=stopped al tracker...")
	left := computeLeft()
	downloaded := length - left
	_, err = sendAnnounce(announce, infoHashEncoded, peerId, listenPort, 0, downloaded, left, "stopped", *hostnameFlag)
	if err != nil {
		fmt.Println("[ERROR] No se pudo enviar stopped:", err)
	} else {
		fmt.Println("[SHUTDOWN] Event=stopped enviado correctamente")
	}

	// Cerrar el listener de conexiones
	fmt.Println("[SHUTDOWN] Cerrando listener...")
	ln.Close()

	// Dar tiempo a las goroutines para terminar
	time.Sleep(500 * time.Millisecond)

	fmt.Println("[SHUTDOWN] Cliente cerrado correctamente. Adiós!")
}
