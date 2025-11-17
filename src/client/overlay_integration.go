package client

import (
	"fmt"
	"src/overlay"
	"strings"
)

// SetupOverlay inicializa el overlay gossip si está habilitado
func SetupOverlay(discoveryMode string, bootstrap string, overlayPort int) *overlay.Overlay {
	var ov *overlay.Overlay
	if discoveryMode == "overlay" {
		// parse bootstrap list
		var peers []string
		if bootstrap != "" {
			for _, p := range strings.Split(bootstrap, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					peers = append(peers, p)
				}
			}
		}
		listenAddr := fmt.Sprintf(":%d", overlayPort)
		ov = overlay.NewOverlay(listenAddr, peers)
		if err := ov.Start(); err != nil {
			fmt.Println("No se pudo iniciar overlay:", err)
			ov = nil
		} else {
			fmt.Println("Overlay gossip iniciado en", listenAddr)
		}
	}

	return ov
}
