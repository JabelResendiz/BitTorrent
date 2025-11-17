// client/cmd/main.go

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"src/client"
	"src/overlay"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Variables de control para manejo de eventos
	var (
		sigChan           = make(chan os.Signal, 1)
		shutdownChan      = make(chan struct{})
		completedChan     = make(chan struct{})
		downloadCompleted bool
		completedMu       sync.Mutex
	)

	var torrentFlag, archivesFlag, hostnameFlag, discoveryFlag, bootstrapFlag string
	var overlayPortFlag int
	torrentFlag, archivesFlag, hostnameFlag, discoveryFlag, bootstrapFlag, overlayPortFlag = client.ParseFlags()

	cfg := client.LoadTorrentMetadata(torrentFlag, archivesFlag)
	// Abrir listener local (puerto asignado automáticamente)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	listenPort := ln.Addr().(*net.TCPAddr).Port
	fmt.Println("Cliente escuchando en puerto:", listenPort)

	var ov *overlay.Overlay
	ov = client.SetupOverlay(discoveryFlag, bootstrapFlag, overlayPortFlag)

	store, mgr, useFinal := client.SetupStorage(cfg)

	client.SetupPieceCompletionHandler(store, cfg, useFinal, completedChan, &completedMu, downloadCompleted)

	computeLeft := client.CreateComputeLeftFunc(store, cfg.FileLength)

	// Enviar announce inicial con event=started
	initialLeft := computeLeft()
	var trackerResponse map[string]interface{}
	trackerInterval := 1800 * time.Second

	if hostnameFlag == "" {
		hostnameFlag = "127.0.0.1"
	}

	providerAddr := fmt.Sprintf("%s:%d", hostnameFlag, listenPort)
	if ov != nil {
		ov.Announce(cfg.InfoHashEncoded, overlay.ProviderMeta{Addr: providerAddr, PeerId: cfg.PeerId, Left: initialLeft})
		fmt.Println("Announced to overlay, left=", initialLeft)
		// keep default trackerInterval
	} else {
		initialLeft := computeLeft()
		trackerResponse, err = client.SendAnnounce(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.PeerId, listenPort, 0, 0, initialLeft, "started", hostnameFlag)
		if err != nil {
			panic(fmt.Errorf("error en announce inicial: %w", err))
		}
		fmt.Println("Tracker responde:", trackerResponse)

		// Hacer scrape para obtener estadísticas del torrent
		client.SendScrape(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.InfoHash)

		// Extraer intervalo del tracker (por defecto 30 minutos)
		if intervalRaw, ok := trackerResponse["interval"].(int64); ok {
			trackerInterval = time.Duration(intervalRaw) * time.Second
			fmt.Printf("Intervalo de announces: %v\n", trackerInterval)
		}
	}

	// trackerResponse, err := client.SendAnnounce(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.PeerId, listenPort, 0, 0, initialLeft, "started", hostnameFlag)
	// if err != nil {
	// 	panic(fmt.Errorf("error en announce inicial: %w", err))
	// }
	// fmt.Println("Tracker responde:", trackerResponse)

	// Hacer scrape para obtener estadísticas del torrent
	//client.SendScrape(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.InfoHash)

	// Extraer intervalo del tracker (por defecto 30 minutos)

	// if intervalRaw, ok := trackerResponse["interval"].(int64); ok {
	// 	trackerInterval = time.Duration(intervalRaw) * time.Second
	// 	fmt.Printf("Intervalo de announces: %v\n", trackerInterval)
	// }

	peerInfo := client.ParsePeersFromOthers(trackerResponse, ov, providerAddr, cfg)
	//peerInfo := client.ParsePeersFromTracker(trackerResponse)

	client.ConnectToPeers(peerInfo, cfg.InfoHash, cfg.PeerId, store, mgr)

	// Aceptar conexiones entrantes
	client.StartListeningForIncomingPeers(ln, cfg.InfoHash, cfg.PeerId, store, mgr)

	// Goroutine: Announces periódicos al tracker
	//client.StartPeriodicAnnounceRoutine(cfg, listenPort, hostnameFlag, computeLeft, shutdownChan, trackerInterval)
	client.StartPeriodicAnnounceRoutineOverlay(cfg, listenPort, hostnameFlag, computeLeft, shutdownChan, trackerInterval, ov, providerAddr)

	// Goroutine: Detectar completación y enviar event=completed
	//client.StartCompletionAnnounceRoutine(completedChan, cfg, listenPort, hostnameFlag)
	client.StartCompletionAnnounceRoutineOverlay(completedChan, cfg, listenPort, hostnameFlag, ov, providerAddr)

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
	//client.SendStoppedAnnounce(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.PeerId, listenPort, cfg.FileLength, computeLeft, hostnameFlag)
	client.SendStoppedAnnounceOverlay(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.PeerId, listenPort, cfg.FileLength, computeLeft, hostnameFlag, ov, providerAddr)

	// Cerrar el listener de conexiones
	fmt.Println("[SHUTDOWN] Cerrando listener...")
	ln.Close()

	// Dar tiempo a las goroutines para terminar
	time.Sleep(500 * time.Millisecond)

	fmt.Println("[SHUTDOWN] Cliente cerrado correctamente. Adiós!")
}
