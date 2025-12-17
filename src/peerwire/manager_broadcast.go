package peerwire

import (
	"fmt"
	"sync"
)

// PieceDownload rastrea el estado de descarga de una pieza desde múltiples peers
type PieceDownload struct {
	pieceIndex       int
	blocksPending    map[int]bool      // bloque index -> true si falta descargar
	blocksInProgress map[int]*PeerConn // bloque index -> peer que lo está descargando
	blocksReceived   map[string]int    // peerAddr -> cantidad de bloques recibidos
}

type Manager struct {
	mu             sync.RWMutex
	peers          map[*PeerConn]struct{}
	store          PieceStore
	pieceDownloads map[int]*PieceDownload // pieceIndex -> estado de descarga
	downloadsMu    sync.Mutex             // protege pieceDownloads
}

func NewManager(store PieceStore) *Manager {
	m := &Manager{
		peers:          make(map[*PeerConn]struct{}),
		store:          store,
		pieceDownloads: make(map[int]*PieceDownload),
	}
	if store != nil {
		store.OnPieceComplete(func(idx int) { m.BroadcastHave(idx) })
	}
	return m
}

func (m *Manager) AddPeer(p *PeerConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[p] = struct{}{}
}

func (m *Manager) RemovePeer(p *PeerConn) {
	m.mu.Lock()
	delete(m.peers, p)
	m.mu.Unlock()

	// Liberar bloques que este peer estaba descargando
	m.downloadsMu.Lock()
	piecesToRetry := make(map[int][]int) // pieceIndex -> lista de bloques a reintentar

	for pieceIndex, pd := range m.pieceDownloads {
		blocksToRetry := []int{}

		// Buscar bloques que este peer estaba descargando
		for blockNum, peer := range pd.blocksInProgress {
			if peer == p {
				blocksToRetry = append(blocksToRetry, blockNum)
			}
		}

		// Liberar esos bloques y devolverlos a pending
		for _, blockNum := range blocksToRetry {
			delete(pd.blocksInProgress, blockNum)
			pd.blocksPending[blockNum] = true
			fmt.Printf("[CLEANUP] Bloque %d de pieza %d liberado por peer desconectado\n", blockNum, pieceIndex)
		}

		if len(blocksToRetry) > 0 {
			piecesToRetry[pieceIndex] = blocksToRetry
		}
	}
	m.downloadsMu.Unlock()

	// Reintentar descargar los bloques liberados desde otros peers
	for pieceIndex, blocks := range piecesToRetry {
		m.retryPendingBlocks(pieceIndex, blocks)
	}
}

func (m *Manager) BroadcastHave(index int) {
	m.mu.RLock()
	peers := make([]*PeerConn, 0, len(m.peers))
	for p := range m.peers {
		peers = append(peers, p)
	}
	m.mu.RUnlock()

	println("Broadcast HAVE a", len(peers), "peers. Pieza", index)
	for _, p := range peers {
		_ = p.SendHave(uint32(index))
	}
}

func (m *Manager) Store() PieceStore { return m.store }

// GetPeerCount retorna el número de peers conectados
func (m *Manager) GetPeerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peers)
}

// HasPeerAddr returns true if there is already a peer with the given remote address (ip:port)
func (m *Manager) HasPeerAddr(addr string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for p := range m.peers {
		if p != nil && p.Conn != nil && p.Conn.RemoteAddr() != nil && p.Conn.RemoteAddr().String() == addr {
			return true
		}
	}
	return false
}

// calculateNumBlocks calcula cuántos bloques tiene una pieza
func (m *Manager) calculateNumBlocks(pieceIndex int) int {
	if m.store == nil {
		return 0
	}
	plen := m.store.PieceLength()
	// última pieza puede ser más corta
	if pieceIndex == m.store.NumPieces()-1 {
		total := m.store.TotalLength()
		plen = int(total - int64(m.store.PieceLength())*int64(m.store.NumPieces()-1))
	}
	numBlocks := (plen + blockLen - 1) / blockLen // ceil division
	return numBlocks
}

// DownloadPieceParallel distribuye bloques de una pieza en Round-Robin entre peers disponibles
func (m *Manager) DownloadPieceParallel(pieceIndex int) {
	if m.store == nil || m.store.HasPiece(pieceIndex) {
		return
	}

	// Verificar si ya se está descargando esta pieza (prevenir race condition)
	m.downloadsMu.Lock()
	if _, alreadyDownloading := m.pieceDownloads[pieceIndex]; alreadyDownloading {
		m.downloadsMu.Unlock()
		println("Pieza", pieceIndex, "ya está siendo descargada, omitiendo solicitud duplicada")
		return
	}
	// Reservar la pieza inmediatamente para evitar duplicados
	m.pieceDownloads[pieceIndex] = &PieceDownload{
		pieceIndex:       pieceIndex,
		blocksPending:    make(map[int]bool),
		blocksInProgress: make(map[int]*PeerConn),
		blocksReceived:   make(map[string]int),
	}
	m.downloadsMu.Unlock()

	// PASO 1: Filtrar peers que tienen esta pieza y están unchoked
	m.mu.RLock()
	availablePeers := []*PeerConn{}
	for peer := range m.peers {
		if peer.RemoteHasPiece(pieceIndex) && !peer.PeerChoking {
			availablePeers = append(availablePeers, peer)
		}
	}
	m.mu.RUnlock()

	if len(availablePeers) == 0 {
		// Limpiar reserva si no hay peers disponibles
		m.downloadsMu.Lock()
		delete(m.pieceDownloads, pieceIndex)
		m.downloadsMu.Unlock()
		println("No hay peers disponibles para pieza", pieceIndex)
		return
	}

	println("Descargando pieza", pieceIndex, "desde", len(availablePeers), "peers en paralelo (Round-Robin)")

	// PASO 2: Inicializar tracking de bloques
	numBlocks := m.calculateNumBlocks(pieceIndex)
	m.downloadsMu.Lock()
	pd := m.pieceDownloads[pieceIndex]
	for i := 0; i < numBlocks; i++ {
		pd.blocksPending[i] = true
	}
	m.downloadsMu.Unlock()

	// PASO 3: Round-Robin - distribuir bloques entre peers
	peerIndex := 0
	plen := m.store.PieceLength()
	if pieceIndex == m.store.NumPieces()-1 {
		total := m.store.TotalLength()
		plen = int(total - int64(m.store.PieceLength())*int64(m.store.NumPieces()-1))
	}

	for blockNum := 0; blockNum < numBlocks; blockNum++ {
		peer := availablePeers[peerIndex%len(availablePeers)]
		offset := blockNum * blockLen

		// Calcular tamaño del bloque (último puede ser menor)
		sz := blockLen
		if offset+sz > plen {
			sz = plen - offset
		}

		// Marcar peer como descargando esta pieza (con lock para thread-safety)
		peer.curPiece = pieceIndex
		peer.downloading = true

		m.downloadsMu.Lock()
		pd.blocksInProgress[blockNum] = peer
		m.downloadsMu.Unlock()

		// Log: Mostrar desde qué peer se solicita el bloque
		peerAddr := "unknown"
		if peer.Conn != nil && peer.Conn.RemoteAddr() != nil {
			peerAddr = peer.Conn.RemoteAddr().String()
		}
		println("  → Solicitando bloque", blockNum, "de pieza", pieceIndex, "a peer", peerAddr)

		// Enviar REQUEST
		peer.SendBlockRequest(uint32(pieceIndex), uint32(offset), uint32(sz))

		peerIndex++
	}
}

// retryPendingBlocks reintenta descargar bloques pendientes de una pieza desde peers disponibles
func (m *Manager) retryPendingBlocks(pieceIndex int, blocks []int) {
	if m.store == nil || m.store.HasPiece(pieceIndex) {
		return
	}

	// Obtener peers disponibles que tienen esta pieza
	m.mu.RLock()
	availablePeers := []*PeerConn{}
	for peer := range m.peers {
		if peer.RemoteHasPiece(pieceIndex) && !peer.PeerChoking {
			availablePeers = append(availablePeers, peer)
		}
	}
	m.mu.RUnlock()

	if len(availablePeers) == 0 {
		fmt.Printf("[RETRY] No hay peers disponibles para reintentar bloques de pieza %d\n", pieceIndex)
		return
	}

	fmt.Printf("[RETRY] Reintentando %d bloques de pieza %d desde %d peers\n", len(blocks), pieceIndex, len(availablePeers))

	plen := m.store.PieceLength()
	if pieceIndex == m.store.NumPieces()-1 {
		total := m.store.TotalLength()
		plen = int(total - int64(m.store.PieceLength())*int64(m.store.NumPieces()-1))
	}

	peerIndex := 0
	for _, blockNum := range blocks {
		peer := availablePeers[peerIndex%len(availablePeers)]
		offset := blockNum * blockLen

		sz := blockLen
		if offset+sz > plen {
			sz = plen - offset
		}

		// Marcar bloque como en progreso
		m.downloadsMu.Lock()
		if pd, exists := m.pieceDownloads[pieceIndex]; exists {
			pd.blocksInProgress[blockNum] = peer
			delete(pd.blocksPending, blockNum)
		}
		m.downloadsMu.Unlock()

		peerAddr := "unknown"
		if peer.Conn != nil && peer.Conn.RemoteAddr() != nil {
			peerAddr = peer.Conn.RemoteAddr().String()
		}
		fmt.Printf("  → [RETRY] Solicitando bloque %d de pieza %d a peer %s\n", blockNum, pieceIndex, peerAddr)

		peer.SendBlockRequest(uint32(pieceIndex), uint32(offset), uint32(sz))
		peerIndex++
	}
}
