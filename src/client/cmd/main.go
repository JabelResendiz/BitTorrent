// client/cmd/main.go

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"src/client"
	"src/overlay"
	"src/utils"
	"strings"
	"sync"
	"syscall"
	"time"
)

var log = utils.NewLogger("CLIENT")

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
		log.Error("No se pudo abrir el listener: %v", err)
		panic(err)
	}

	listenPort := ln.Addr().(*net.TCPAddr).Port
	log.Info("Cliente escuchando en puerto: %d", listenPort)

	ov := client.SetupOverlay(discoveryFlag, bootstrapFlag, overlayPortFlag)
	if ov != nil {
		log.Info("=== Modo de descubrimiento: OVERLAY/GOSSIP (distribuido) ===")
	} else {
		log.Info("=== Modo de descubrimiento: TRACKER (centralizado) ===")
	}

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
		// ov.Announce(cfg.InfoHashEncoded, overlay.ProviderMeta{Addr: providerAddr, PeerId: cfg.PeerId, Left: initialLeft})
		// fmt.Println("Announced to overlay, left=", initialLeft)

		// construir lista initialPeers: SOLO nodos remotos
		initialPeers := []string{}
		if bootstrapFlag != "" {
			for _, p := range strings.Split(bootstrapFlag, ",") {
				p = strings.TrimSpace(p)
				if p != "" && p != providerAddr {
					initialPeers = append(initialPeers, p)
				}
			}
		}

		// Hacer discovery síncrono antes de anunciar
		ttlDepth := 3
		if err := ov.Discover(cfg.InfoHashEncoded, initialPeers, ttlDepth); err != nil {
			log.Warn("Overlay discovery returned error: %v", err)
		} else {
			log.Info("Overlay discovery completed; store has providers for infohash")
		}

		// Ahora sí nos anunciamos al overlay
		ov.Announce(cfg.InfoHashEncoded, overlay.ProviderMeta{
			Addr:   providerAddr,
			PeerId: cfg.PeerId,
			Left:   initialLeft,
		})
		log.Info("Announced to overlay, left=%d", initialLeft)

	} else {
		initialLeft := computeLeft()
		trackerResponse, err = client.SendAnnounce(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.PeerId, listenPort, 0, 0, initialLeft, "started", hostnameFlag)
		if err != nil {
			log.Error("Error en announce inicial: %v", err)
			panic(err)
		}
		log.Info("Tracker responde: %+v", trackerResponse)

		// Hacer scrape para obtener estadísticas del torrent
		client.SendScrape(cfg.AnnounceURL, cfg.InfoHashEncoded, cfg.InfoHash)

		// Extraer intervalo del tracker (por defecto 30 minutos)
		if intervalRaw, ok := trackerResponse["interval"].(int64); ok {
			trackerInterval = time.Duration(intervalRaw) * time.Second
			log.Info("Intervalo de announces: %v", trackerInterval)
		}
	}

	peerInfo := client.ParsePeersFromOthers(trackerResponse, ov, providerAddr, cfg)

	client.ConnectToPeers(peerInfo, cfg.InfoHash, cfg.PeerId, store, mgr)

	// Aceptar conexiones entrantes
	client.StartListeningForIncomingPeers(ln, cfg.InfoHash, cfg.PeerId, store, mgr)

	// Goroutine: Announces periódicos (tracker o overlay según modo)
	client.StartPeriodicAnnounceRoutineOverlay(cfg, listenPort, hostnameFlag, computeLeft, shutdownChan, trackerInterval, ov, providerAddr)

	// Goroutine: Detectar completación y enviar event=completed
	client.StartCompletionAnnounceRoutineOverlay(completedChan, cfg, listenPort, hostnameFlag, ov, providerAddr)

	// Configurar captura de señales del sistema
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	// Loop principal: esperar señal de terminación
	log.Info("=== Cliente BitTorrent ejecutándose ===")
	log.Info("Presiona Ctrl+C para detener el cliente")
	log.Info("Escuchando en puerto: %d", listenPort)
	log.Info("Announces cada: %v", trackerInterval)

	sig := <-sigChan

	// Shutdown limpio
	log.Warn("=== Señal %v recibida, iniciando shutdown limpio ===", sig)

	// Notificar a todas las goroutines que deben detenerse
	close(shutdownChan)

	// Enviar stopped (tracker o overlay según modo)
	client.SendStoppedAnnounceOverlay(
		cfg.AnnounceURL,
		cfg.InfoHashEncoded,
		cfg.PeerId,
		listenPort,
		cfg.FileLength,
		computeLeft,
		hostnameFlag,
		ov,
		providerAddr,
	)

	// Cerrar el listener de conexiones
	log.Warn("Cerrando listener...")
	ln.Close()

	// Dar tiempo a las goroutines para terminar
	time.Sleep(500 * time.Millisecond)

	log.Info("Cliente cerrado correctamente. Adiós!")
}
