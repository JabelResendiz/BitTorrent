package client

import (
	"fmt"
	"src/overlay"
	"src/peerwire"
	"time"
)

func StartCompletionAnnounceRoutine(
	completedChan <-chan struct{},
	cfg *ClientConfig,
	listenPort int,
	hostnameFlag string,
) {

	go func() {
		<-completedChan
		fmt.Println("[INFO] Enviando event=completed al tracker...")

		_, err := SendAnnounceWithFailover(
			cfg,
			listenPort,
			0,              // uploaded
			cfg.FileLength, // downloaded
			0,              // left
			"completed",
			hostnameFlag,
		)

		if err != nil {
			fmt.Println("[ERROR] No se pudo enviar completed:", err)
		} else {
			fmt.Println("[INFO] Ahora soy un seeder completo")
		}
	}()
}

func StartCompletionAnnounceRoutineOverlay(
	completedChan <-chan struct{},
	cfg *ClientConfig,
	listenPort int,
	hostnameFlag string,
	ov *overlay.Overlay,
	providerAddr string,
) {

	go func() {
		<-completedChan

		if ov != nil {
			fmt.Println("[INFO] Enviando event=completed al overlay...")
			ov.Announce(cfg.InfoHashEncoded, overlay.ProviderMeta{Addr: providerAddr, PeerId: cfg.PeerId, Left: 0})
			fmt.Println("[INFO] Ahora soy un seeder completo (overlay)")
		} else {
			fmt.Println("[INFO] Enviando event=completed al tracker...")
			_, err := SendAnnounceWithFailover(
				cfg,
				listenPort,
				0,              // uploaded
				cfg.FileLength, // downloaded
				0,              // left
				"completed",
				hostnameFlag,
			)

			if err != nil {
				fmt.Println("[ERROR] No se pudo enviar completed:", err)
			} else {
				fmt.Println("[INFO] Ahora soy un seeder completo")
			}
		}
	}()
}

// StartPeriodicAnnounceRoutine sends periodic announces to the tracker
func StartPeriodicAnnounceRoutine(
	cfg *ClientConfig,
	listenPort int,
	hostname string,
	computeLeft func() int64,
	shutdownChan <-chan struct{},
	trackerInterval time.Duration,
	infoHash [20]byte,
	peerId string,
	store interface{},
	mgr interface{},
) {
	go func() {
		ticker := time.NewTicker(trackerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				left := computeLeft()

				trackerResponse, err := SendAnnounceWithFailover(
					cfg,
					listenPort,
					0,    // uploaded
					0,    // downloaded
					left, // left (actualizado)
					"",   // event vacío
					hostname,
				)

				if err != nil {
					fmt.Println("[ERROR] Announce periódico fallido:", err)
				} else {
					fmt.Println("[INFO] Announce periódico enviado")

					// Procesar nuevos peers de la respuesta
					peerInfo := ParsePeersFromOthers(trackerResponse, nil, "", cfg)
					if len(peerInfo) > 0 {
						fmt.Printf("[INFO] Conectando a %d peers nuevos del announce periódico\n", len(peerInfo))
						if diskStore, ok := store.(*peerwire.DiskPieceStore); ok {
							if manager, ok := mgr.(*peerwire.Manager); ok {
								ConnectToPeers(peerInfo, infoHash, peerId, diskStore, manager)
							}
						}
					}
				}

			case <-shutdownChan:
				fmt.Println("[INFO] Periodic announce detenido")
				return
			}
		}
	}()
}

func StartPeriodicAnnounceRoutineOverlay(
	cfg *ClientConfig,
	listenPort int,
	hostname string,
	computeLeft func() int64,
	shutdownChan <-chan struct{},
	trackerInterval time.Duration,
	ov *overlay.Overlay,
	providerAddr string,
	infoHash [20]byte,
	peerId string,
	store interface{},
	mgr interface{},
) {
	go func() {
		ticker := time.NewTicker(trackerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				left := computeLeft()

				if ov != nil {
					ov.Announce(cfg.InfoHashEncoded, overlay.ProviderMeta{Addr: providerAddr, PeerId: cfg.PeerId, Left: left})
					fmt.Println("[INFO] Announce periódico enviado (overlay)")

					// Obtener y conectar a nuevos peers del overlay
					peerInfo := ParsePeersFromOthers(nil, ov, providerAddr, cfg)
					if len(peerInfo) > 0 {
						fmt.Printf("[INFO] Conectando a %d peers nuevos del overlay\n", len(peerInfo))
						if diskStore, ok := store.(*peerwire.DiskPieceStore); ok {
							if manager, ok := mgr.(*peerwire.Manager); ok {
								ConnectToPeers(peerInfo, infoHash, peerId, diskStore, manager)
							}
						}
					}
				} else {
					trackerResponse, err := SendAnnounceWithFailover(
						cfg,
						listenPort,
						0, // uploaded
						0, // downloaded
						left,
						"",
						hostname,
					)

					if err != nil {
						fmt.Println("[ERROR] Announce periódico fallido (tracker):", err)
					} else {
						fmt.Println("[INFO] Announce periódico enviado (tracker)")

						// Procesar nuevos peers de la respuesta
						peerInfo := ParsePeersFromOthers(trackerResponse, nil, "", cfg)
						if len(peerInfo) > 0 {
							fmt.Printf("[INFO] Conectando a %d peers nuevos del announce periódico\n", len(peerInfo))
							if diskStore, ok := store.(*peerwire.DiskPieceStore); ok {
								if manager, ok := mgr.(*peerwire.Manager); ok {
									ConnectToPeers(peerInfo, infoHash, peerId, diskStore, manager)
								}
							}
						}
					}
				}

			case <-shutdownChan:
				fmt.Println("[INFO] Periodic announce detenido")
				return
			}
		}
	}()
}
