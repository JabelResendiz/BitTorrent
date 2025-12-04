package overlay

import "time"

func(o *Overlay) PeriodicHealthCheck() {
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			o.checkDeadPeers()
		case <-o.stopCh:
			ticker.Stop()
			return
		}
	}
}


func (o *Overlay) checkDeadPeers() {
	now := time.Now().Unix()
	timeout := int64(20)

	all := o.Store.AllProviders()

	for infoHash, providers := range all {
		alive := []ProviderMeta{}

		for _, pm := range providers {
			if now-pm.LastSeen < timeout {
				alive = append(alive,pm)
			} else {
				o.Logger.Warn("Peer muerto: %s",pm.Addr)
			}
		}

		o.Store.Replace(infoHash,alive)
	}
}