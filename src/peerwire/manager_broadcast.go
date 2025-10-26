package peerwire

import "sync"

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
	defer m.mu.Unlock()
	delete(m.peers, p)
}

func (m *Manager) BroadcastHave(index int) {
	m.mu.RLock()
	peers := make([]*PeerConn, 0, len(m.peers))
	for p := range m.peers {
		peers = append(peers, p)
	}
	m.mu.RUnlock()
	// debug log
	// Note: keep it simple; avoid import cycles by not using log package here.
	// The client will see this on stdout.
	println("Broadcast HAVE a", len(peers), "peers. Pieza", index)
	for _, p := range peers {
		_ = p.SendHave(uint32(index))
	}
}

func (m *Manager) Store() PieceStore { return m.store }

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
