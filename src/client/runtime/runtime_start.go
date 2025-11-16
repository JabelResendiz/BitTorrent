// package runtime

// import (
// 	"fmt"
// 	"src/client"
// )

// func (rt *ClientRuntime) Start() error {
// 	fmt.Println("[BOOT] Iniciando cliente BitTorrent...")

// 	// Announce inicial
// 	initialLeft := rt.ComputeLeft()
// 	tr, err := client.SendAnnounce(rt.Cfg.AnnounceURL, rt.Cfg.InfoHashEncoded,
// 		rt.Cfg.PeerId, rt.ListenPort, 0, 0, initialLeft, "started", rt.HostnameFlag)

// 	if err != nil {
// 		return fmt.Errorf("announce inicial falló: %w", err)
// 	}

// 	fmt.Println("[BOOT] Tracker responde:", tr)

// 	// Scrape
// 	go client.SendScrape(rt.Cfg.AnnounceURL, rt.Cfg.InfoHashEncoded, rt.Cfg.InfoHash)

// 	// Peers del tracker
// 	peers := client.ParsePeersFromTracker(tr)

// 	// Conexiones salientes
// 	client.ConnectToPeers(peers, rt.Cfg.InfoHash, rt.Cfg.PeerId, rt.Store, rt.Manager)

// 	// Listener de conexiones entrantes
// 	client.StartListeningForIncomingPeers(rt.Listener, rt.Cfg.InfoHash, rt.Cfg.PeerId, rt.Store, rt.Manager)

// 	// Intervalo del tracker
// 	interval := client.ParseTrackerInterval(tr)

// 	// Rutina: announces periódicos
// 	rt.startPeriodicAnnounces(interval)

// 	// Rutina: notificar completado
// 	rt.startCompletionRoutine()

// 	return nil
// }
