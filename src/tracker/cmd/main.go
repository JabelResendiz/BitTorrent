package main

//tracker/cmd/main.go

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"src/tracker"
	"strings"
	"time"
)

func main() {
	// Flags de configuración del tracker:
	// -listen: dirección de escucha HTTP (por defecto ":8080")
	// -interval: segundos para el campo "interval" en respuestas y base para expiración
	// -maxpeers: límite de peers a devolver en /announce
	// -sync-listen: dirección de escucha para sincronización entre trackers
	// -sync-peers: lista de direcciones de otros trackers (separados por coma)
	// -sync-interval: intervalo de sincronización en segundos
	listen := flag.String("listen", ":8080", "address to listen, e.g. :8080")
	interval := flag.Int("interval", 1800, "announce interval in seconds")
	maxPeers := flag.Int("maxpeers", 50, "max peers per response")
	syncListen := flag.String("sync-listen", ":9090", "address to listen for sync messages, e.g. :9090")
	syncPeersStr := flag.String("sync-peers", "", "comma-separated list of remote tracker addresses for sync, e.g. tracker2:9090,tracker3:9090")
	syncInterval := flag.Int("sync-interval", 15, "sync interval in seconds")
	flag.Parse()

	// Obtener hostname del contenedor como node-id (automático con Docker)
	nodeID, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// Usar hostname como nombre del archivo de datos (automático)
	dataPath := fmt.Sprintf("/data/%s_data.json", nodeID)

	// Parsear lista de peers remotos
	var remotePeers []string
	if *syncPeersStr != "" {
		remotePeers = strings.Split(*syncPeersStr, ",")
		// Limpiar espacios
		for i := range remotePeers {
			remotePeers[i] = strings.TrimSpace(remotePeers[i])
		}
	}

	// Construye el tracker con Interval y PeerTimeout (2×interval por defecto),
	// límite de peers por respuesta, ruta para persistencia, node-id y peers remotos.
	t := tracker.New(
		time.Duration(*interval)*time.Second,
		time.Duration(*interval*2)*time.Second,
		*maxPeers,
		dataPath,
		nodeID,
		remotePeers,
	)

	log.Printf("Tracker node-id: %s, data: %s", nodeID, dataPath)

	// Carga estado previo desde disco si existe.
	if err := t.LoadFromFile(); err != nil {
		log.Fatalf("load failed: %v", err)
	}

	// Iniciar sincronización distribuida si hay peers remotos
	if len(remotePeers) > 0 {
		log.Printf("Starting distributed sync with %d peers", len(remotePeers))

		// Log de estado de seguridad
		tracker.LogSecurityStatus()

		// Iniciar listener de sincronización
		if err := t.StartSyncListener(*syncListen); err != nil {
			log.Fatalf("failed to start sync listener: %v", err)
		}

		// Iniciar manager de sincronización periódica
		t.StartSyncManager(time.Duration(*syncInterval) * time.Second)
	}

	// Registra el handler /announce del tracker.
	http.HandleFunc("/announce", t.AnnounceHandler)
	// Registra el handler /scrape del tracker.
	http.HandleFunc("/scrape", t.ScrapeHandler)

	// GC loop
	// Bucle en background que expira peers inactivos periódicamente y persiste
	// los cambios cuando elimina alguno.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if exp := t.GC(); exp > 0 {
				_ = t.SaveToFile()
				log.Printf("gc expired %d peers", exp)
			}
		}
	}()

	// // IP del tracker y nombre DNS que quieres usar
	// trackerIP := "127.0.0.1"
	// trackerName := "tracker"
	// dnsAPI := "127.0.0.1:6969"

	// // Registrar en DNS
	// if err := dns.RegisterInDNS(trackerName, trackerIP, dnsAPI); err != nil {
	// 	log.Printf("failed to register tracker in DNS: %v", err)
	// } else {
	// 	log.Printf("tracker registered in DNS as %s -> %s", trackerName, trackerIP)
	// }

	// Arranca el servidor HTTP del tracker.
	log.Printf("tracker listening on %s interval=%ds", *listen, *interval)
	if err := http.ListenAndServe(*listen, nil); err != nil {
		log.Fatal(err)
	}
}
