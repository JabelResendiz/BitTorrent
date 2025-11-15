package peerwire

// PiecePicker provides a minimal strategy to choose which piece to download next from a peer.
// Phase 1: simple first-needed strategy (can be upgraded to rarest-first later).
type PiecePicker struct{}

func NewPiecePicker() *PiecePicker { return &PiecePicker{} }

// NextPieceFor returns the index of a piece that we need and the peer has, or -1 if none.
func (pp *PiecePicker) NextPieceFor(p *PeerConn, store PieceStore) int {
	if p == nil || store == nil {
		return -1
	}
	n := store.NumPieces()
	for i := 0; i < n; i++ {
		if !store.HasPiece(i) && p.RemoteHasPiece(i) {
			return i
		}
	}
	return -1
}
