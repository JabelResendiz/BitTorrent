package overlay

import (
	"encoding/json"
	"net"
	"sort"
	"time"
	"src/utils"
)

// message types used on the wire
type wireMsg struct {
	Type      string         `json:"type"`
	InfoHash  string         `json:"info_hash,omitempty"`
	Providers []ProviderMeta `json:"providers,omitempty"`
	Limit     int            `json:"limit,omitempty"`
}

// Overlay is the main entry point: holds the store, peers and runs gossip listener
type Overlay struct {
	Store      *Store
	peers      []string
	listenAddr string
	stopCh     chan struct{}
	Logger     *utils.Logger
}

// NewOverlay crea un overlay con TTL por defecto de 90s
func NewOverlay(listenAddr string, peers []string) *Overlay {
	s := NewStore(90 * time.Second)
	return &Overlay{
		Store: s, 
		peers: peers, 
		listenAddr: listenAddr, 
		stopCh: make(chan struct{}),
		Logger: utils.NewLogger("Overlay")}
}

// Start inicia el listener TCP y el loop de gossip periódico
func (o *Overlay) Start() error {
	o.Logger.Info("Iniciando Overlay en %s con peers %v", o.listenAddr, o.peers)
	
	ln, err := net.Listen("tcp", o.listenAddr)
	if err != nil {
		o.Logger.Error("Fallo escuchando en %s: %v", o.listenAddr, err)
        return err
	}
	go o.ServeListener(ln)
	go o.PeriodicGossip() // cada 8 seg
	go o.PeriodicHealthCheck() // cada 10 seg

	o.Logger.Info("Overlay iniciado correctamente")
	return nil
}

// Stop detiene el overlay
func (o *Overlay) Stop() {
	select {
	case <-o.stopCh:
		return
	default:
		close(o.stopCh)
	}
}

// // acepta conexiones entrantes mientras no se cierre el overlay
// func (o *Overlay) serveListener(ln net.Listener) {
// 	defer ln.Close()
// 	for {
// 		conn, err := ln.Accept()
// 		if err != nil {
// 			select {
// 			case <-o.stopCh:
// 				return
// 			default:
// 			}
// 			continue
// 		}
// 		go o.handleConn(conn)
// 	}
// }

// func (o *Overlay) handleConn(conn net.Conn) {
// 	defer conn.Close()
// 	dec := json.NewDecoder(conn)
// 	var m wireMsg
// 	if err := dec.Decode(&m); err != nil {
// 		// ignore decode errors
// 		return
// 	}
// 	switch strings.ToLower(m.Type) {
// 	case "gossip", "announce":
// 		if m.InfoHash != "" && len(m.Providers) > 0 {
// 			o.Store.Merge(m.InfoHash, m.Providers)
// 		}
// 		// no reply required
// 	case "lookup":
// 		// respond with local providers for requested infoHash
// 		if m.InfoHash == "" {
// 			return
// 		}
// 		provs := o.Store.Lookup(m.InfoHash, m.Limit)
// 		enc := json.NewEncoder(conn)
// 		// reply with providers array
// 		_ = enc.Encode(provs)
// 	default:
// 		return
// 	}
// }

// // periodicGossip envia nuestro estado a peers conocidos periódicamente
// func (o *Overlay) periodicGossip() {
// 	ticker := time.NewTicker(8 * time.Second)
// 	defer ticker.Stop()
// 	for {
// 		select {
// 		case <-ticker.C:
// 			o.gossipOnce()
// 		case <-o.stopCh:
// 			return
// 		}
// 	}
// }

// // gossipOnce: por simplicidad envia por cada infoHash full provider list a los peers
// func (o *Overlay) gossipOnce() {
// 	o.Store.mu.RLock()
// 	infohashes := make([]string, 0, len(o.Store.records))
// 	for ih := range o.Store.records {
// 		infohashes = append(infohashes, ih)
// 	}
// 	o.Store.mu.RUnlock()

// 	for _, peer := range o.peers {
// 		for _, ih := range infohashes {
// 			b, err := o.Store.ToJSON(ih)
// 			if err != nil {
// 				continue
// 			}
// 			msg := wireMsg{Type: "gossip", InfoHash: ih}
// 			// attach providers
// 			_ = json.Unmarshal(b, &msg.Providers)
// 			go sendWireMsg(peer, msg)
// 		}
// 	}
// }

func sendWireMsg(addr string, msg wireMsg) {
	conn, err := net.DialTimeout("tcp", addr, 1200*time.Millisecond)
	if err != nil {
		return
	}
	defer conn.Close()
	enc := json.NewEncoder(conn)
	_ = enc.Encode(msg)
	// don't wait for reply
}

// func (o *Overlay) periodicHealthCheck() {
// 	ticker := time.NewTicker(10 * time.Second)
// 	for {
// 		select {
// 		case <-ticker.C:
// 			o.checkDeadPeers()
// 		case <-o.stopCh:
// 			ticker.Stop()
// 			return
// 		}
// 	}
// }

// func (o *Overlay) checkDeadPeers() {
// 	now := time.Now().Unix()
// 	timeout := int64(20) // peer muerto si no responde por 20s

// 	all := o.Store.AllProviders() // tú defines este helper

// 	for infoHash, providers := range all {
// 		alive := []ProviderMeta{}

// 		for _, pm := range providers {
// 			if now-pm.LastSeen < timeout {
// 				alive = append(alive, pm)
// 			} else {
// 				fmt.Println("[OVERLAY] Peer muerto:", pm.Addr)
// 			}
// 		}

// 		o.Store.Replace(infoHash, alive)
// 	}
// }

// Announce locally registers the provider and also tries to push to peers
func (o *Overlay) Announce(infoHash string, p ProviderMeta) {
	_ = o.Store.Announce(infoHash, p)
	// fire-and-forget push to bootstrap peers
	msg := wireMsg{Type: "announce", InfoHash: infoHash, Providers: []ProviderMeta{p}}
	for _, peer := range o.peers {
		go sendWireMsg(peer, msg)
	}
}

// Lookup queries local store and also asks up to 'fanout' peers for providers
func (o *Overlay) Lookup(infoHash string, limit int) []ProviderMeta {
	// local
	outMap := make(map[string]ProviderMeta)
	local := o.Store.Lookup(infoHash, limit)
	for _, p := range local {
		outMap[p.Addr] = p
	}

	// query up to 3 peers
	fanout := 3
	queried := 0
	for _, peer := range o.peers {
		if queried >= fanout {
			break
		}
		queried++
		provs := queryPeerLookup(peer, infoHash, limit)
		for _, p := range provs {
			if _, ok := outMap[p.Addr]; !ok {
				outMap[p.Addr] = p
			}
		}
	}
	// convert to slice and sort
	out := make([]ProviderMeta, 0, len(outMap))
	for _, v := range outMap {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen > out[j].LastSeen })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func queryPeerLookup(addr, infoHash string, limit int) []ProviderMeta {
	var out []ProviderMeta
	conn, err := net.DialTimeout("tcp", addr, 800*time.Millisecond)
	if err != nil {
		return out
	}
	defer conn.Close()
	enc := json.NewEncoder(conn)
	msg := wireMsg{Type: "lookup", InfoHash: infoHash, Limit: limit}
	if err := enc.Encode(msg); err != nil {
		return out
	}
	// read reply
	dec := json.NewDecoder(conn)
	_ = dec.Decode(&out)
	return out
}
