package peerwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const blockLen = 16 * 1024

// requestNextBlocks requests up to a small window of blocks for the active piece
func (p *PeerConn) requestNextBlocks(piece int) {
	if p.manager == nil || p.manager.Store() == nil {
		return
	}
	store := p.manager.Store()
	plen := store.PieceLength()
	// last piece size
	if piece == store.NumPieces()-1 {
		total := store.TotalLength()
		plen = int(total - int64(store.PieceLength())*int64(store.NumPieces()-1))
	}
	// if we already have the piece skip
	if store.HasPiece(piece) {
		return
	}
	// establish per-peer state
	if p.curPiece != piece {
		p.curPiece = piece
		p.curOffset = 0
	}
	begin := p.curOffset
	// size to request this time
	sz := blockLen
	if begin+sz > plen {
		sz = plen - begin
	}
	if sz <= 0 {
		return
	}
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(piece))
	binary.Write(&buf, binary.BigEndian, uint32(begin))
	binary.Write(&buf, binary.BigEndian, uint32(sz))
	_ = p.SendMessage(MsgRequest, buf.Bytes())
	p.downloading = true
}

func (p *PeerConn) ReadLoop() {
	for {
		id, payload, err := p.ReadMessage()
		if err != nil {
			fmt.Println("Error con peer:", err)
			p.Close()
			return
		}

		p.handleMessage(id, payload)
	}
}

// actualiza estado segun el tipo de mensaje
func (p *PeerConn) handleMessage(id byte, payload []byte) {
	switch id {
	case MsgChoke:
		p.PeerChoking = true
	case MsgInterested:
		p.PeerInterested = true
		_ = p.SendMessage(MsgUnchoke, nil)
	case MsgNotInterested:
		p.PeerInterested = false
	case MsgUnchoke:
		p.PeerChoking = false
		fmt.Println("Peer te unchokeo. Buscando pieza a solicitar...")
		if p.manager != nil && p.manager.Store() != nil {
			if !p.downloading {
				picker := NewPiecePicker()
				piece := picker.NextPieceFor(p, p.manager.Store())
				if piece >= 0 {
					p.manager.DownloadPieceParallel(piece)
				} else {
					fmt.Println("Nada que pedir a este peer")
				}
			}
		}
	case MsgHave:
		index := binary.BigEndian.Uint32(payload)
		fmt.Println("Peer tiene pieza:", index)
		if p.manager != nil && p.manager.Store() != nil {
			n := p.manager.Store().NumPieces()
			if int(index) < n {
				exp := (n + 7) / 8
				if len(p.remoteBF) != exp {
					p.remoteBF = make([]byte, exp)
				}
				byteIdx := int(index) / 8
				bit := 7 - (int(index) % 8)
				p.remoteBF[byteIdx] |= (1 << uint(bit))
			}
		}
	case MsgBitfiled:
		if p.manager != nil && p.manager.Store() != nil {
			exp := (p.manager.Store().NumPieces() + 7) / 8
			if len(payload) != exp {
				fmt.Println("Bitfield inválido: tamaño", len(payload), "esperado", exp)
				return
			}
		}
		p.UpdateRemoteBitfield(payload)
		fmt.Println("Bitfield inicial recibido")
		if p.manager != nil && p.manager.Store() != nil {
			haveInterest := false
			n := p.manager.Store().NumPieces()
			for i := 0; i < n; i++ {
				if p.RemoteHasPiece(i) && !p.manager.Store().HasPiece(i) {
					haveInterest = true
					break
				}
			}
			if haveInterest {
				p.SendMessage(MsgInterested, nil)
			}
		}
	case MsgPort:
		fmt.Println("Puerto abierto")
	case MsgPiece:
		if len(payload) < 8 {
			fmt.Println("Piece demasiado corta")
			return
		}
		index := binary.BigEndian.Uint32(payload[0:4])
		begin := binary.BigEndian.Uint32(payload[4:8])
		block := payload[8:]

		// Log: Mostrar desde qué peer se recibió el bloque
		peerAddr := "unknown"
		if p.Conn != nil && p.Conn.RemoteAddr() != nil {
			peerAddr = p.Conn.RemoteAddr().String()
		}
		blockNum := int(begin) / blockLen
		fmt.Printf("✓ Recibido bloque %d de pieza %d desde peer %s (offset %d, tamaño %d bytes)\n",
			blockNum, index, peerAddr, begin, len(block))

		// Guardar bloque en el storage
		if p.manager != nil && p.manager.Store() != nil {
			if _, err := p.manager.Store().WriteBlock(int(index), int(begin), block); err != nil {
				fmt.Println("Error guardando bloque:", err)
				return
			}

			// Marcar bloque como recibido en el tracking Round-Robin
			blockNum := int(begin) / blockLen
			p.manager.downloadsMu.Lock()
			if pd, exists := p.manager.pieceDownloads[int(index)]; exists {
				pd.blocksReceived[peerAddr]++

				delete(pd.blocksPending, blockNum)
				delete(pd.blocksInProgress, blockNum)

				// Verificar si la pieza está completa
				if len(pd.blocksPending) == 0 {
					fmt.Printf("\n═══════════════════════════════════════════════\n")
					fmt.Printf("✓ Pieza %d completada (Round-Robin)\n", index)
					fmt.Printf("═══════════════════════════════════════════════\n")
					fmt.Printf("Distribución de bloques por peer:\n")
					totalBlocks := 0
					for pAddr, count := range pd.blocksReceived {
						fmt.Printf("  • Peer %s: %d bloques\n", pAddr, count)
						totalBlocks += count
					}
					fmt.Printf("Total: %d bloques\n", totalBlocks)
					fmt.Printf("═══════════════════════════════════════════════\n\n")

					// Limpiar tracking
					delete(p.manager.pieceDownloads, int(index))
				}
			}
			p.manager.downloadsMu.Unlock()

			// Si la pieza está verificada y completa, buscar siguiente pieza
			if p.manager.Store().HasPiece(int(index)) {
				picker := NewPiecePicker()
				nxt := picker.NextPieceFor(p, p.manager.Store())
				if nxt >= 0 {
					p.curPiece = nxt
					p.curOffset = 0
					p.downloading = false
					p.manager.DownloadPieceParallel(nxt)
				} else {
					p.downloading = false
					fmt.Println("Descarga completada, no hay más piezas")
				}
			}
		}
	case MsgRequest:
		if len(payload) != 12 {
			return
		}
		idx := binary.BigEndian.Uint32(payload[0:4])
		rbegin := binary.BigEndian.Uint32(payload[4:8])
		rlen := binary.BigEndian.Uint32(payload[8:12])
		if p.manager == nil || p.manager.Store() == nil {
			return
		}
		data, err := p.manager.Store().ReadBlock(int(idx), int(rbegin), int(rlen))
		if err != nil {
			return
		}
		_ = p.SendPiece(idx, rbegin, data)
	case 255:
		//ignorar
	default:
		fmt.Println("Mensaje desconocido", id)
	}
}
