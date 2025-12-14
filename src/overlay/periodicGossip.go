package overlay

import (
	"encoding/json"
	"time"
)

// periodicGossip envia nuestro estado a peers conocidos periódicamente
func (o *Overlay) PeriodicGossip() {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			o.gossipOnce()
		case <-o.stopCh:
			return
		}
	}
}

// gossipOnce: por simplicidad envia por cada infoHash full provider list a los peers
func (o *Overlay) gossipOnce() {
	o.Store.mu.RLock()
	infohashes := make([]string, 0, len(o.Store.records))
	for ih := range o.Store.records {
		infohashes = append(infohashes, ih)
	}
	o.Store.mu.RUnlock()

	for _, peer := range o.peers {
		for _, ih := range infohashes {
			b, err := o.Store.ToJSON(ih)
			if err != nil {
				continue
			}
			msg := wireMsg{Type: "gossip", InfoHash: ih}
			// attach providers
			_ = json.Unmarshal(b, &msg.Providers)
			go sendWireMsg(peer, msg)
		}
	}
}
