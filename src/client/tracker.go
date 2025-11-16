package client

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"src/bencode"
	"strings"
	"time"
)

// var httpclient = dns.ResolveCustomHTTPClient("127.0.0.1:8053")

func GeneratePeerId() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("-JC0001-%s", hex.EncodeToString(buf))
}

func SendAnnounce(announceURL, infoHashEncoded, peerId string, port int,
	uploaded, downloaded, left int64, event string, hostname string) (map[string]interface{}, error) {

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

	if event != "" {
		params.Set("event", event)
	}

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

	// Decodificar la respuesta
	trackerResponse, err := bencode.Decode(resp.Body)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error decodificando respuesta: %w", err)
	}

	// verificar failure reason
	if failureReason, ok := trackerResponse["failure reason"].(string); ok {
		return trackerResponse, fmt.Errorf("tracker error: %s", failureReason)
	}

	return trackerResponse, nil
}

// envia una peticion scrape al tracker y muestra las estadisticas
func SendScrape(announceURL, infoHashEncoded string, infoHash [20]byte) {
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

	// Extraer y mostrar estadisticas
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
