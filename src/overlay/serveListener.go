package overlay

import (
	"encoding/json"
	"net"
	"strings"
)

// acepta conexiones entrantes mientras no se cierre el overlay
func (o *Overlay) ServeListener(ln net.Listener) {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-o.stopCh:
				return
			default:
			}
			continue
		}
		go o.handleConn(conn)
	}
}

func (o *Overlay) handleConn(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	var m wireMsg
	if err := dec.Decode(&m); err != nil {
		// ignore decode errors
		return
	}
	switch strings.ToLower(m.Type) {
	case "gossip", "announce":
		if m.InfoHash != "" && len(m.Providers) > 0 {
			o.Store.Merge(m.InfoHash, m.Providers)
		}
		// no reply required
	case "lookup":
		// respond with local providers for requested infoHash
		if m.InfoHash == "" {
			return
		}
		provs := o.Store.Lookup(m.InfoHash, m.Limit)
		enc := json.NewEncoder(conn)
		// reply with providers array
		_ = enc.Encode(provs)
	default:
		return
	}
}