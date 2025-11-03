package main

//tracker/cmd/main.go

import (
	"flag"
	"log"
	"net/http"
	"time"
	"src/tracker"
)

func main() {
	// Flags de configuración del tracker:
	// -listen: dirección de escucha HTTP (por defecto ":8080")
	// -interval: segundos para el campo "interval" en respuestas y base para expiración
	// -data: ruta del archivo JSON donde se persiste el estado
	// -maxpeers: límite de peers a devolver en /announce
	listen := flag.String("listen", ":8080", "address to listen, e.g. :8080")
	interval := flag.Int("interval", 1800, "announce interval in seconds")
	data := flag.String("data", "tracker_data.json", "path to JSON persistence file")
	maxPeers := flag.Int("maxpeers", 50, "max peers per response")
	flag.Parse()

	// Construye el tracker con Interval y PeerTimeout (2×interval por defecto),
	// límite de peers por respuesta y ruta para persistencia.
	t := tracker.New(time.Duration(*interval)*time.Second, time.Duration(*interval*2)*time.Second, *maxPeers, *data)
	// Carga estado previo desde disco si existe.
	if err := t.LoadFromFile(); err != nil {
		log.Fatalf("load failed: %v", err)
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
			if exp := t.GC(time.Now()); exp > 0 {
				_ = t.SaveToFile()
				log.Printf("gc expired %d peers", exp)
			}
		}
	}()

	// Arranca el servidor HTTP del tracker.
	log.Printf("tracker listening on %s interval=%ds data=%s", *listen, *interval, *data)
	if err := http.ListenAndServe(*listen, nil); err != nil {
		log.Fatal(err)
	}
}
