
// gossip.go
// this module implements a simple gossip protocol for synchronizing DNS recrods
//between mutiple peers. Each node runs a TCP server to receive updates and 
// periodically sends its own records to all know peers

package internal

import (
	"encoding/json"
	"net"
	"time"
)

var gossiplog = NewLogger("GOSSIP")


func StartGossip(peers []string, s * Store){
	
	// listen for incoming connections for other peers
	go func() {
		ln, err := net.Listen("tcp",":5300")

		if err != nil {
			gossiplog.Error("Fialed to start TCP listener :%v", err)
			return
		}

		gossiplog.Info("TCP server listening on :5300")

		for {
			conn, err := ln.Accept()
			if err != nil {
				gossiplog.Warn("Failed to accept connection: %v", err)
				continue
			}

			gossiplog.Info("Accepted connection from %s", conn.RemoteAddr())
			go handleConn(conn, s)
		}
	}()
	
	// Perodically send its state to other peers
	go func(){
		for {
			payload := GossipMessage{Type:"update", Records: s.List()}

			b, _ := json.Marshal(payload)

			for _, peer := range peers {
				conn, err := net.DialTimeout("tcp",peer,time.Second)

				if err != nil {
					gossiplog.Warn("Failed to connect to peer %s: %v", peer, err)
					continue
				}

				gossiplog.Debug("Sending update to peer %s (%d records)", peer, len(payload.Records))
				conn.Write(b)
				conn.Close()
			}
			time.Sleep(10*time.Second)
		}
	}()
}


// Read the JSON from a peer and update the records in the store
func handleConn(conn net.Conn, s* Store){
	defer conn.Close()

	var msg GossipMessage

	dec := json.NewDecoder(conn)

	if err := dec.Decode(&msg) ; err != nil {
		gossiplog.Warn("Failed to decode message from %s: %v", conn.RemoteAddr(), err)
		return
	}

	gossiplog.Info("Received %d records from %s", len(msg.Records), conn.RemoteAddr())
	for _, r := range msg.Records {
		s.Add(r)
		gossiplog.Debug("Updated record: %s -> %s", r.Name, r.IP)
	}
}