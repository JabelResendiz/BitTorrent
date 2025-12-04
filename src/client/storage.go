package client

import (
	"fmt"
	"net"
	"os"
	"src/overlay"
	"src/peerwire"
	"sync"
	"time"
)

type ComputeLeftFunc func() int64

func CreateComputeLeftFunc(store *peerwire.DiskPieceStore, fileLength int64) ComputeLeftFunc {
	return func() int64 {
		var have int64
		num := store.NumPieces()
		for i := 0; i < num; i++ {
			if store.HasPiece(i) {
				if i == num-1 {
					have += fileLength - int64(store.PieceLength())*int64(num-1)
				} else {
					have += int64(store.PieceLength())
				}
			}
		}
		if have > fileLength {
			have = fileLength
		}
		return fileLength - have
	}
}

func SetupStorage(cfg *ClientConfig) (*peerwire.DiskPieceStore, *peerwire.Manager, bool) {
	tempPath, finalPath := cfg.GetStoragePaths()

	useFinal := false
	usePartResume := false

	if st, err := os.Stat(finalPath); err == nil && st.Size() == cfg.FileLength {
		useFinal = true
	} else if st, err := os.Stat(tempPath); err == nil && st.Size() == cfg.FileLength {
		usePartResume = true
	}

	var store *peerwire.DiskPieceStore
	var err error

	if useFinal {
		store, err = peerwire.NewDiskPieceStoreWithMode(finalPath, int(cfg.PieceLength), cfg.FileLength, false)
	} else if usePartResume {
		store, err = peerwire.NewDiskPieceStoreWithMode(tempPath, int(cfg.PieceLength), cfg.FileLength, false)
	} else {
		store, err = peerwire.NewDiskPieceStore(tempPath, int(cfg.PieceLength), cfg.FileLength)
	}

	if err != nil {
		panic(err)
	}

	mgr := peerwire.NewManager(store)

	// Verificacion SHA-1 por pieza si tenemos los hashes esperados
	if len(cfg.ExpectedHashes) == store.NumPieces() {
		store.SetExpectedHashes(cfg.ExpectedHashes)
	}

	// Si existe archivo final o .part previo,intentar marcar piezas copletas por SHA-1
	if (useFinal || usePartResume) && len(cfg.ExpectedHashes) == store.NumPieces() {
		if err := store.ScanAndMarkComplete(); err != nil {
			fmt.Println("No se pudo escanear archivo existente para marcar piezas:", err)
		}
	}

	return store, mgr, useFinal
}

func SetupPieceCompletionHandler(store *peerwire.DiskPieceStore, cfg *ClientConfig,
	useFinal bool, completedChan chan struct{}, completedMu *sync.Mutex, downloadCompleted bool) {

	tempPath, finalPath := cfg.GetStoragePaths()

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

				// notificar que la descarga se completo
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
}

func ConnectToPeers(peers []PeerInfo, infoHash [20]byte, peerId string,
	store *peerwire.DiskPieceStore, mgr *peerwire.Manager) {

	seen := make(map[string]struct{})

	for _, peerInfo := range peers {
		if _, dup := seen[peerInfo.Addr]; dup {
			fmt.Printf("Peer duplicado omitido: %s\n", peerInfo.Addr)
			continue
		}
		seen[peerInfo.Addr] = struct{}{}
		fmt.Printf("Peer: %s\n", peerInfo.Addr)

		// Probe: verificar que el puerto realmente escucha antes de intentar handshake
		// (evita errores con providers stale del overlay)
		conn, err := net.DialTimeout("tcp", peerInfo.Addr, 2*time.Second)
		if err != nil {
			fmt.Printf("  [SKIP] Peer inaccesible: %v\n", err)
			continue
		}
		conn.Close() // solo verificamos, cerramos la conexión de prueba

		var peerIdBytes [20]byte
		copy(peerIdBytes[:], []byte(peerId))

		pc, err := peerwire.NewPeerConn(peerInfo.Addr, infoHash, peerIdBytes)
		if err != nil {
			fmt.Println("Error creando PeerConn:", err)
			continue
		}
		//defer pc.Close()

		pc.BindManager(mgr)

		if err := pc.Handshake(); err != nil {
			fmt.Println("Handshake fallido:", err)
			pc.Close()
			continue
		}

		fmt.Println("Conectado al peer, handshake OK")
		_ = pc.SendBitfield(store.Bitfield())
		pc.SendMessage(peerwire.MsgInterested, nil)

		//go pc.ReadLoop()

		go func(pc *peerwire.PeerConn) {
			defer pc.Close() // cierre correcto
			pc.ReadLoop()    // loop de lectura
		}(pc)
	}
}

func SendStoppedAnnounce(announceURL, infoHashEncoded, peerId string, listenPort int,
	fileLength int64, computeLeft ComputeLeftFunc, hostname string) {

	fmt.Println("[SHUTDOWN] Enviando event=stopped al tracker...")
	left := computeLeft()
	downloaded := fileLength - left

	var err error
	_, err = SendAnnounce(announceURL, infoHashEncoded, peerId,
		listenPort, 0, downloaded, left, "stopped", hostname)

	if err != nil {
		fmt.Println("[ERROR] No se pudo enviar stopped:", err)
	} else {
		fmt.Println("[SHUTDOWN] Event=stopped enviado correctamente")
	}
}

func SendStoppedAnnounceOverlay(announceURL, infoHashEncoded, peerId string, listenPort int,
	fileLength int64, computeLeft ComputeLeftFunc, hostname string,
	ov *overlay.Overlay, providerAddr string) {

	fmt.Println("[SHUTDOWN] Enviando event=stopped al tracker...")
	left := computeLeft()
	downloaded := fileLength - left

	var err error

	if ov != nil {
		ov.Announce(infoHashEncoded, overlay.ProviderMeta{Addr: providerAddr, PeerId: peerId, Left: left})
		fmt.Println("[SHUTDOWN] Anuncio enviado al overlay")
	} else {
		_, err = SendAnnounce(announceURL, infoHashEncoded, peerId,
			listenPort, 0, downloaded, left, "stopped", hostname)

		if err != nil {
			fmt.Println("[ERROR] No se pudo enviar stopped:", err)
		} else {
			fmt.Println("[SHUTDOWN] Event=stopped enviado correctamente")
		}
	}

}
