package tracker

// tracker/sync_messages.go
// Definición de mensajes para sincronización entre trackers distribuidos.

// SyncMessage es el mensaje que se envía periódicamente entre trackers
// para sincronizar el estado de los swarms.
type SyncMessage struct {
	FromNodeID string                      `json:"from_node_id"` // ID del tracker emisor
	Timestamp  HLC                         `json:"timestamp"`    // HLC del mensaje
	Swarms     map[string]map[string]*Peer `json:"swarms"`       // infoHash -> peerID -> Peer
}

// NewSyncMessage crea un nuevo mensaje de sincronización con el estado actual del tracker.
func (t *Tracker) NewSyncMessage() *SyncMessage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Actualizar HLC para el evento de creación de mensaje
	t.hlc.Update(nil)

	// Copiar todos los swarms y peers
	swarms := make(map[string]map[string]*Peer)
	for infoHash, swarm := range t.Torrents {
		peers := make(map[string]*Peer)
		for peerID, peer := range swarm.Peers {
			// Copiar peer (incluyendo tombstones)
			peerCopy := &Peer{
				PeerIDHex: peer.PeerIDHex,
				IP:        peer.IP,
				Port:      peer.Port,
				LastSeen:  peer.LastSeen,
				Completed: peer.Completed,
				HostName:  peer.HostName,
				Deleted:   peer.Deleted,
			}
			peers[peerID] = peerCopy
		}
		swarms[infoHash] = peers
	}

	return &SyncMessage{
		FromNodeID: t.nodeID,
		Timestamp:  t.hlc.Clone(),
		Swarms:     swarms,
	}
}
