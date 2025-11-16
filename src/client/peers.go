package client

import (
	"encoding/binary"
	"fmt"
)

type PeerInfo struct {
	Addr string
}

func ParsePeersFromTracker(trackerResponse map[string]interface{}) []PeerInfo {
	var peerAddrs []string

	if peersRaw, ok := trackerResponse["peers"].(string); ok {
		// Formato compact: 6 bytes por peer (4 IP + 2 puerto)
		data := []byte(peersRaw)
		for i := 0; i < len(data); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
			port := binary.BigEndian.Uint16(data[i+4 : i+6])
			addr := fmt.Sprintf("%s:%d", ip, port)
			peerAddrs = append(peerAddrs, addr)
		}
	} else if peersList, ok := trackerResponse["peers"].([]interface{}); ok {
		for _, peerRaw := range peersList {
			if peerDict, ok := peerRaw.(map[string]interface{}); ok {
				var ip string
				var port int64

				if ipVal, ok := peerDict["ip"].(string); ok {
					ip = ipVal
				}
				if portVal, ok := peerDict["port"].(int64); ok {
					port = portVal
				}

				if ip != "" && port > 0 {
					addr := fmt.Sprintf("%s:%d", ip, port)
					peerAddrs = append(peerAddrs, addr)
				}
			}
		}
	}

	peers := make([]PeerInfo, len(peerAddrs))
	for i, addr := range peerAddrs {
		peers[i] = PeerInfo{Addr: addr}
	}
	return peers
}
