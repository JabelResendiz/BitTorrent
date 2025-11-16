package client

import (
	"fmt"
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

		_, err := SendAnnounce(
			cfg.AnnounceURL,
			cfg.InfoHashEncoded,
			cfg.PeerId,
			listenPort,
			0,
			cfg.FileLength,
			0,
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

// StartPeriodicAnnounceRoutine sends periodic announces to the tracker
func StartPeriodicAnnounceRoutine(
	cfg *ClientConfig,
	listenPort int,
	hostname string,
	computeLeft func() int64,
	shutdownChan <-chan struct{},
	trackerInterval time.Duration,
) {
	go func() {
		ticker := time.NewTicker(trackerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				left := computeLeft()

				_, err := SendAnnounce(
					cfg.AnnounceURL,
					cfg.InfoHashEncoded,
					cfg.PeerId,
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
				}

			case <-shutdownChan:
				fmt.Println("[INFO] Periodic announce detenido")
				return
			}
		}
	}()
}
