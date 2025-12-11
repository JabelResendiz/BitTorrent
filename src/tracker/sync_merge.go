package tracker

// tracker/sync_merge.go
// Implementación de merge de estado usando LWW (Last Write Wins) con tombstones.

import (
	"log"
)

// MergeSwarms procesa un mensaje de sincronización y hace merge del estado remoto
// con el estado local usando la estrategia LWW (Last Write Wins) basada en HLC.
func (t *Tracker) MergeSwarms(msg *SyncMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Actualizar HLC local con el timestamp del mensaje
	t.hlc.Update(&msg.Timestamp)

	log.Printf("[SYNC] Merging swarms from node %s", msg.FromNodeID)

	// Procesar cada swarm del mensaje
	for infoHash, remotePeers := range msg.Swarms {
		// Asegurar que el swarm existe localmente
		localSwarm := t.getOrCreateSwarm(infoHash)

		// Procesar cada peer del swarm remoto
		for peerID, remotePeer := range remotePeers {
			t.mergePeer(infoHash, localSwarm, peerID, remotePeer)
		}
	}
}

// mergePeer hace merge de un peer individual usando LWW y tombstone resurrection.
func (t *Tracker) mergePeer(infoHash string, localSwarm *Swarm, peerID string, remotePeer *Peer) {
	localPeer := localSwarm.Peers[peerID]

	if localPeer == nil {
		// Caso 1: El peer no existe localmente, agregarlo (incluso si es tombstone)
		localSwarm.Peers[peerID] = &Peer{
			PeerIDHex: remotePeer.PeerIDHex,
			IP:        remotePeer.IP,
			Port:      remotePeer.Port,
			LastSeen:  remotePeer.LastSeen,
			Completed: remotePeer.Completed,
			HostName:  remotePeer.HostName,
			Deleted:   remotePeer.Deleted,
		}
		log.Printf("[SYNC] Added new peer %s to swarm %s (deleted=%v)", peerID, infoHash[:8], remotePeer.Deleted)
		return
	}

	// Caso 2: El peer existe localmente, comparar timestamps (LWW)
	if remotePeer.LastSeen.After(localPeer.LastSeen) {
		// El peer remoto es más reciente

		// Subcaso 2a: Tombstone resurrection
		if localPeer.Deleted && !remotePeer.Deleted {
			// Local está eliminado pero remoto está activo y es más reciente -> resucitar
			localPeer.Deleted = false
			localPeer.IP = remotePeer.IP
			localPeer.Port = remotePeer.Port
			localPeer.LastSeen = remotePeer.LastSeen
			localPeer.Completed = remotePeer.Completed
			localPeer.HostName = remotePeer.HostName
			log.Printf("[SYNC] Resurrected peer %s in swarm %s", peerID, infoHash[:8])
			return
		}

		// Subcaso 2b: Actualización normal o propagación de tombstone
		localPeer.IP = remotePeer.IP
		localPeer.Port = remotePeer.Port
		localPeer.LastSeen = remotePeer.LastSeen
		localPeer.Completed = remotePeer.Completed
		localPeer.HostName = remotePeer.HostName
		localPeer.Deleted = remotePeer.Deleted

		if remotePeer.Deleted {
			log.Printf("[SYNC] Updated peer %s in swarm %s to tombstone", peerID, infoHash[:8])
		} else {
			log.Printf("[SYNC] Updated peer %s in swarm %s", peerID, infoHash[:8])
		}
		return
	}

	// Caso 3: El peer local es más reciente o igual -> ignorar el remoto
	log.Printf("[SYNC] Ignored older update for peer %s in swarm %s", peerID, infoHash[:8])
}
