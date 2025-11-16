// package runtime

// import (
// 	"fmt"
// 	"time"
// 	"src/client"
// )

// func (rt *ClientRuntime) Shutdown() {
// 	fmt.Println("[SHUTDOWN] Notificando goroutines...")
// 	close(rt.ShutdownChan)

// 	fmt.Println("[SHUTDOWN] Enviando event=stopped...")
// 	client.SendStoppedAnnounce(rt.Cfg.AnnounceURL, rt.Cfg.InfoHashEncoded,
// 		rt.Cfg.PeerId, rt.ListenPort, rt.Cfg.FileLength, rt.ComputeLeft, rt.HostnameFlag)

// 	fmt.Println("[SHUTDOWN] Cerrando listener...")
// 	rt.Listener.Close()

// 	time.Sleep(300 * time.Millisecond)
// 	fmt.Println("[SHUTDOWN] Cliente cerrado correctamente.")
// }
