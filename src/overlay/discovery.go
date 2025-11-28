package overlay

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

func checkNodeAlive(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}

	_ = conn.Close()

	return true
}

func (o *Overlay) Discover(infoHash string, initialPeers []string, ttl int) error {

	fmt.Printf("[DISCOVER] Iniciando discovery para infohash %s con TTL %d y bootstraps: %v\n", infoHash, ttl, initialPeers)

	if len(initialPeers) == 0 {
		return fmt.Errorf("Discover: initialPeers vacío")
	}
	seen := make(map[string]struct{})
	type node struct {
		addr  string
		depth int
	}
	// cola BFS
	queue := []node{}
	for _, p := range initialPeers {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		queue = append(queue, node{addr: p, depth: 0})
	}

	// ensure bootstrap initial peers are added to store (so quedan persistidos)
	now := time.Now().Unix()
	for _, p := range initialPeers {
		if p == "" {
			continue
		}
		// agrega con PeerId vacío si no tenemos peerId; LastSeen se ajusta en Announce si fuera necesario
		pm := ProviderMeta{Addr: p, PeerId: "", Left: 0, LastSeen: now}
		fmt.Printf("%v", pm)
		o.Store.Merge(infoHash, []ProviderMeta{pm})
	}

	// BFS
	var mu sync.Mutex
	var wg sync.WaitGroup
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]

		// saltar si ya visto o depth > ttl
		if _, ok := seen[n.addr]; ok {
			continue
		}
		if n.depth > ttl {
			continue
		}

		// marcar como visto
		seen[n.addr] = struct{}{}

		// comprobar si el nodo responde
		if !checkNodeAlive(n.addr, 800*time.Millisecond) {
			// nodos que no responden no se exploran
			continue
		}

		// consultamos su lookup (en goroutine para paralelizar)
		wg.Add(1)
		go func(addr string, depth int) {
			defer wg.Done()
			// pedimos muchos providers para maximizar discovery
			provs := queryPeerLookup(addr, infoHash, 50)
			if len(provs) == 0 {
				return
			}

			// insertar providers en store (Merge ya es thread-safe)
			o.Store.Merge(infoHash, provs)

			// añadir nuevos nodos a la cola (si no están vistos)
			mu.Lock()
			for _, p := range provs {
				if _, ok := seen[p.Addr]; !ok && p.Addr != "" && depth+1 <= ttl {
					queue = append(queue, node{addr: p.Addr, depth: depth + 1})
				}
			}
			mu.Unlock()
		}(n.addr, n.depth)

	}
	// esperar a que terminen las consultas
	wg.Wait()

	// último chequeo: si no hay providers en store para el infohash devolvemos error
	finalList := o.Store.Lookup(infoHash, 1)
	if len(finalList) == 0 {
		return fmt.Errorf("Discover: no providers encontrados (quizá TTL o bootstraps fallaron)")
	}

	return nil
}
