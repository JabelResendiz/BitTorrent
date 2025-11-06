package main

import (
    "os"
    "strings"
	"src/dns/internal"
)

func main() {

	log := internal.NewLogger("MAIN")

	log.Info("Starting DNS Service")

    peersEnv := os.Getenv("PEERS")
    peers := []string{}
    if peersEnv != "" {
        peers = strings.Split(peersEnv, ",")
		log.Info("Loaded peers : %v", peers)
    } else{
		log.Warn("No peers specified; running standalone")
	}

	// apiPort := os.Getenv("API_PORT")
	// if apiPort == "" {
	// 	apiPort = "8080" 
	// }
	// dnsPort := os.Getenv("DNS_PORT")
	// if dnsPort == "" {
	// 	dnsPort = "8053"
	// }

    s := internal.New()
    go  func() {
		log.Info("Starting UDP resolver at :8053")
		internal.StartUDP(s, ":8053")
		
	}()
	
	go  func() {
		log.Info("Starting gossip service")
		internal.StartGossip(peers, s)
	}()
	
	log.Info("Starting API on :8080")
	internal.Start(s)
}
