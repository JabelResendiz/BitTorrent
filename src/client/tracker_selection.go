package client

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// TrackerLatency almacena la latencia de un tracker
type TrackerLatency struct {
	URL     string
	Latency time.Duration
	Index   int
}

// PingTracker mide la latencia de un tracker haciendo una petición HEAD o GET rápida
func PingTracker(trackerURL string, timeout time.Duration) (time.Duration, error) {
	// Construir URL base del tracker (sin /announce)
	baseURL := strings.TrimSuffix(trackerURL, "/announce")

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	start := time.Now()

	// Intentar HEAD primero (más rápido)
	resp, err := client.Head(baseURL)
	if err != nil {
		// Si HEAD falla, intentar GET
		resp, err = client.Get(baseURL)
		if err != nil {
			return 0, err
		}
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	return latency, nil
}

// SelectClosestTracker mide la latencia de todos los trackers y retorna el índice del más rápido
func SelectClosestTracker(trackerURLs []string) int {
	if len(trackerURLs) == 0 {
		return 0
	}

	if len(trackerURLs) == 1 {
		fmt.Println("[TRACKER] Solo hay un tracker disponible")
		return 0
	}

	fmt.Printf("[TRACKER] Midiendo latencia de %d trackers...\n", len(trackerURLs))

	latencies := make([]TrackerLatency, 0, len(trackerURLs))
	timeout := 3 * time.Second

	for i, url := range trackerURLs {
		latency, err := PingTracker(url, timeout)
		if err != nil {
			fmt.Printf("  [%d] %s - ERROR: %v\n", i, url, err)
			// Asignar latencia muy alta si falla
			latencies = append(latencies, TrackerLatency{
				URL:     url,
				Latency: 999 * time.Second,
				Index:   i,
			})
		} else {
			fmt.Printf("  [%d] %s - %v\n", i, url, latency)
			latencies = append(latencies, TrackerLatency{
				URL:     url,
				Latency: latency,
				Index:   i,
			})
		}
	}

	// Ordenar por latencia (menor primero)
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i].Latency < latencies[j].Latency
	})

	// Retornar el índice del tracker más rápido
	closestIdx := latencies[0].Index
	fmt.Printf("[TRACKER] Tracker más cercano seleccionado: %s (latencia: %v)\n",
		latencies[0].URL, latencies[0].Latency)

	// Reordenar el slice de URLs poniendo los más rápidos primero
	reorderedURLs := make([]string, len(trackerURLs))
	for i, tl := range latencies {
		reorderedURLs[i] = tl.URL
	}

	// Actualizar el orden (se hará en el caller)
	return closestIdx
}

// SelectAndReorderTrackers mide latencias y reordena la lista poniendo el más rápido primero
func SelectAndReorderTrackers(cfg *ClientConfig) {
	if len(cfg.AnnounceURLs) <= 1 {
		fmt.Println("[TRACKER] Solo hay un tracker, no es necesario reordenar")
		return
	}

	fmt.Println("[TRACKER] Seleccionando tracker más cercano...")

	latencies := make([]TrackerLatency, 0, len(cfg.AnnounceURLs))
	timeout := 3 * time.Second

	for i, url := range cfg.AnnounceURLs {
		latency, err := PingTracker(url, timeout)
		if err != nil {
			fmt.Printf("  [%d] %s - ERROR: %v\n", i, url, err)
			latencies = append(latencies, TrackerLatency{
				URL:     url,
				Latency: 999 * time.Second,
				Index:   i,
			})
		} else {
			fmt.Printf("  [%d] %s - %v\n", i, url, latency)
			latencies = append(latencies, TrackerLatency{
				URL:     url,
				Latency: latency,
				Index:   i,
			})
		}
	}

	// Ordenar por latencia (menor primero)
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i].Latency < latencies[j].Latency
	})

	// Reordenar el slice de URLs poniendo los más rápidos primero
	reorderedURLs := make([]string, len(cfg.AnnounceURLs))
	for i, tl := range latencies {
		reorderedURLs[i] = tl.URL
	}

	cfg.AnnounceURLs = reorderedURLs
	cfg.CurrentTrackerIdx = 0 // El primero es ahora el más rápido

	fmt.Printf("[TRACKER] Tracker seleccionado: %s (latencia: %v)\n",
		cfg.AnnounceURLs[0], latencies[0].Latency)
	fmt.Println("[TRACKER] Orden de failover:")
	for i, url := range cfg.AnnounceURLs {
		fmt.Printf("  [%d] %s\n", i+1, url)
	}
}
