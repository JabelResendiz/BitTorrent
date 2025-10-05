package tracker

import (
	"encoding/binary"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"src/bencode"
)

// AnnounceHandler handles GET /announce minimal params: info_hash, peer_id, port
// AnnounceHandler valida los parámetros mínimos (info_hash, peer_id, port),
// registra/actualiza el peer en el swarm correspondiente y responde con un
// diccionario bencode que incluye el intervalo (interval) y la lista de peers
// en formato compacto IPv4 (6 bytes por peer: 4 de IP + 2 de puerto).
func (t *Tracker) AnnounceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.RawQuery

	infoHash, err := raw20(q, "info_hash")
	if err != nil {
		t.failure(w, "missing or invalid info_hash")
		return
	}
	peerID, err := raw20(q, "peer_id")
	if err != nil {
		t.failure(w, "missing or invalid peer_id")
		return
	}

	vals, _ := url.ParseQuery(q)
	port64, _ := strconv.ParseInt(vals.Get("port"), 10, 32)
	if port64 <= 0 || port64 > 65535 {
		t.failure(w, "invalid port")
		return
	}

	// Parámetros adicionales que el cliente envía; se aceptan y en parte se usan
	uploaded, _ := strconv.ParseInt(vals.Get("uploaded"), 10, 64)
	downloaded, _ := strconv.ParseInt(vals.Get("downloaded"), 10, 64)
	left, _ := strconv.ParseInt(vals.Get("left"), 10, 64)
	if uploaded < 0 || downloaded < 0 || left < 0 {
		t.failure(w, "invalid counters")
		return
	}
	event := vals.Get("event")
	_ = vals.Get("compact") // leemos pero siempre respondemos en formato compacto IPv4
	numwant := t.MaxPeersResp
	if nw, err := strconv.Atoi(vals.Get("numwant")); err == nil && nw >= 0 {
		if nw < numwant {
			numwant = nw
		}
	}

	ip := clientIP(r, vals.Get("ip"))
	if ip == nil || ip.To4() == nil {
		t.failure(w, "ipv4 required")
		return
	}

	infoHex, _ := Bytes20ToHex(infoHash)
	peerHex, _ := Bytes20ToHex(peerID)
	now := time.Now()

	log.Printf("announce from %s ih=%s pid=%s port=%d", ip.String(), infoHex[:8], peerHex[:8], port64)

	// Determinar si el peer es seeder
	completed := left == 0

	// event handling: stopped => eliminar; completed/started/empty => alta/refresh
	switch event {
	case "stopped":
		_ = t.SaveOnChange(func() { t.RemovePeer(infoHex, peerHex) })
	default:
		_ = t.SaveOnChange(func() { t.AddPeer(infoHex, peerHex, ip, uint16(port64), completed, now) })
	}

	// Build peer list excluding requester
	peers := t.GetPeers(infoHex, peerHex, numwant)
	compactResp := compactPeers(peers)

	// Contadores de seeds/leechers (complete/incomplete) para compatibilidad
	comp, incomp := t.CountPeers(infoHex)
	reply := map[string]interface{}{
		"interval":   int64(t.Interval.Seconds()),
		"peers":      compactResp,
		"complete":   int64(comp),
		"incomplete": int64(incomp),
	}
	data := bencode.Encode(relySafe(reply))
	w.Header().Set("Content-Type", "application/x-bittorrent")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// relySafe es un placeholder por si se quisiera sanear/normalizar el mapa de
// respuesta antes de codificarlo. Actualmente devuelve el mismo mapa.
func relySafe(m map[string]interface{}) map[string]interface{} { return m }

// failure envía una respuesta de error bencodeada con la clave "failure reason"
// y código HTTP 400, además de registrar el motivo en el log.
func (t *Tracker) failure(w http.ResponseWriter, reason string) {
	log.Printf("failure: %s", reason)
	data := bencode.Encode(map[string]interface{}{"failure reason": reason})
	w.Header().Set("Content-Type", "application/x-bittorrent")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(data)
}

// compactPeers construye la representación "compact" de la lista de peers,
// concatenando por cada peer 4 bytes de IPv4 y 2 bytes del puerto en big-endian.
func compactPeers(peers []*Peer) string {
	b := make([]byte, 0, len(peers)*6)
	for _, p := range peers {
		ip := net.ParseIP(p.IP).To4()
		if ip == nil {
			continue
		}
		b = append(b, ip[0], ip[1], ip[2], ip[3])
		var port [2]byte
		binary.BigEndian.PutUint16(port[:], p.Port)
		b = append(b, port[0], port[1])
	}
	return string(b)
}

// raw20 extrae un parámetro de la query cruda, aplica percent-unescape y valida
// que el resultado tenga exactamente 20 bytes (como info_hash o peer_id).
func raw20(rawQuery, key string) ([]byte, error) {
	for _, part := range strings.Split(rawQuery, "&") {
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || kv[0] != key {
			continue
		}
		s, err := url.QueryUnescape(kv[1])
		if err != nil {
			return nil, err
		}
		if len(s) != 20 {
			return nil, errors.New("must be 20 bytes")
		}
		return []byte(s), nil
	}
	return nil, errors.New("missing")
}

// clientIP determina la IP del cliente. Si se provee override, intenta usarla;
// de lo contrario toma la IP de la conexión entrante (RemoteAddr).
func clientIP(r *http.Request, override string) net.IP {
	if override != "" {
		if ip := net.ParseIP(override); ip != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	return net.ParseIP(host)
}
