// package runtime

// import (
// 	"net"
// 	"src/client"
// 	"src/peerwire"
// )

// type ClientRuntime struct {
// 	Cfg        *client.ClientConfig
// 	Store      *peerwire.DiskPieceStore
// 	Manager    *peerwire.Manager
// 	Listener   net.Listener
// 	ListenPort int

// 	ComputeLeft   func() int64
// 	ShutdownChan  chan struct{}
// 	CompletedChan chan struct{}
// 	HostnameFlag  string
// }

// func NewRuntime(cfg *client.ClientConfig, hostnameFlag string, store *peerwire.DiskPieceStore,
// 	mgr *peerwire.Manager, ln net.Listener, computeLeft func() int64) *ClientRuntime {

// 	return &ClientRuntime{
// 		Cfg:           cfg,
// 		Store:         store,
// 		Manager:       mgr,
// 		Listener:      ln,
// 		ListenPort:    ln.Addr().(*net.TCPAddr).Port,
// 		ComputeLeft:   computeLeft,
// 		ShutdownChan:  make(chan struct{}),
// 		CompletedChan: make(chan struct{}),
// 		HostnameFlag:  hostnameFlag,
// 	}
// }
