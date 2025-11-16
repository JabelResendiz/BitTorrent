// package runtime

// import (
// 	"fmt"
// 	"src/client"
// 	"time"
// )

// func (rt *ClientRuntime) startPeriodicAnnounces(interval time.Duration) {
// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				left := rt.ComputeLeft()
// 				client.SendAnnounce(rt.Cfg.AnnounceURL, rt.Cfg.InfoHashEncoded,
// 					rt.Cfg.PeerId, rt.ListenPort, 0, 0, left, "", rt.HostnameFlag)

// 			case <-rt.ShutdownChan:
// 				return
// 			}
// 		}
// 	}()
// }

// func (rt *ClientRuntime) startCompletionRoutine() {
// 	go func() {
// 		<-rt.CompletedChan
// 		fmt.Println("[COMPLETE] Enviando event=completed al tracker...")

// 		client.SendAnnounce(rt.Cfg.AnnounceURL, rt.Cfg.InfoHashEncoded, rt.Cfg.PeerId,
// 			rt.ListenPort, 0, rt.Cfg.FileLength, 0, "completed", rt.HostnameFlag)
// 	}()
// }
