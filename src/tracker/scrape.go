package tracker

import (
	"net/http"
	"net/url"

	"src/bencode"
)

// ScrapeHandler implements GET /scrape
// Admite uno o varios parámetros info_hash (20 bytes percent-encoded).
// Responde un diccionario bencode con clave "files" cuyo valor es otro
// diccionario indexado por la clave binaria info_hash (20 bytes), y como valor
// para cada torrent un diccionario con: complete, incomplete, downloaded.
// Nota: usamos downloaded=0 en esta versión.
func (t *Tracker) ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parseamos la query completa para saber si pidieron torrents específicos.
	raw := r.URL.RawQuery
	hashes := raw20multi(raw, "info_hash")

	// files: map con clave binaria (string de 20 bytes) -> stats
	files := make(map[string]interface{})

	if len(hashes) == 0 {
		// En esta implementación, si no se especifican info_hash devolvemos
		// un diccionario vacío (files:{}), dado que no conservamos la forma
		// binaria original de las claves para todos los torrents.
	} else {
		for _, ihRaw := range hashes {
			// Convertimos a hex para buscar el swarm y contar peers
			ihHex, err := Bytes20ToHex(ihRaw)
			if err != nil {
				// Si algo falla con un hash, lo ignoramos
				continue
			}
			comp, incomp := t.CountPeers(ihHex)
			files[string(ihRaw)] = map[string]interface{}{
				"complete":   int64(comp),
				"incomplete": int64(incomp),
				"downloaded": int64(0),
			}
		}
	}

	reply := map[string]interface{}{
		"files": files,
	}
	data := bencode.Encode(reply)
	w.Header().Set("Content-Type", "application/x-bittorrent")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// raw20multi devuelve todas las ocurrencias de una clave en la query cruda
// como slices de 20 bytes tras percent-unescape; ignora entradas inválidas.
func raw20multi(rawQuery, key string) [][]byte {
	vals, _ := url.ParseQuery(rawQuery)
	vs := vals[key]
	if len(vs) == 0 {
		return nil
	}
	res := make([][]byte, 0, len(vs))
	for _, v := range vs {
		// url.ParseQuery ya decodifica percent-encoding
		if len(v) == 20 {
			res = append(res, []byte(v))
		}
	}
	return res
}
