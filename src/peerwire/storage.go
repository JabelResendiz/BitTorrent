package peerwire

import (
	"errors"
	"os"
	"sync"
)

// PieceStore defines the storage contract for pieces/blocks
type PieceStore interface {
	NumPieces() int
	PieceLength() int
	TotalLength() int64

	Bitfield() []byte
	HasPiece(i int) bool

	WriteBlock(piece int, begin int, data []byte) (completed bool, err error)
	ReadBlock(piece int, begin int, length int) ([]byte, error)

	OnPieceComplete(cb func(piece int))
}

// DiskPieceStore is a simple file-backed implementation
type DiskPieceStore struct {
	f           *os.File
	mu          sync.RWMutex
	pieceLength int
	totalLength int64
	numPieces   int

	bitfield  []byte
	completed []bool
	received  []int64 // naive accounting per piece

	cbs []func(int)
}

func NewDiskPieceStore(path string, pieceLength int, totalLength int64) (*DiskPieceStore, error) {
	if pieceLength <= 0 || totalLength <= 0 {
		return nil, errors.New("invalid lengths")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if err := f.Truncate(totalLength); err != nil {
		f.Close()
		return nil, err
	}
	numPieces := int((totalLength + int64(pieceLength) - 1) / int64(pieceLength))
	return &DiskPieceStore{
		f:           f,
		pieceLength: pieceLength,
		totalLength: totalLength,
		numPieces:   numPieces,
		bitfield:    make([]byte, (numPieces+7)/8),
		completed:   make([]bool, numPieces),
		received:    make([]int64, numPieces),
	}, nil
}

func (s *DiskPieceStore) NumPieces() int     { return s.numPieces }
func (s *DiskPieceStore) PieceLength() int   { return s.pieceLength }
func (s *DiskPieceStore) TotalLength() int64 { return s.totalLength }

func (s *DiskPieceStore) Bitfield() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]byte, len(s.bitfield))
	copy(out, s.bitfield)
	return out
}

func (s *DiskPieceStore) HasPiece(i int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i < 0 || i >= s.numPieces {
		return false
	}
	byteIdx := i / 8
	bit := 7 - (i % 8)
	return (s.bitfield[byteIdx] & (1 << uint(bit))) != 0
}

func (s *DiskPieceStore) pieceSize(i int) int64 {
	if i < 0 || i >= s.numPieces {
		return 0
	}
	if i == s.numPieces-1 {
		return s.totalLength - int64(s.pieceLength*(s.numPieces-1))
	}
	return int64(s.pieceLength)
}

func (s *DiskPieceStore) markComplete(i int) {
	byteIdx := i / 8
	bit := 7 - (i % 8)
	s.bitfield[byteIdx] |= (1 << uint(bit))
	s.completed[i] = true
}

func (s *DiskPieceStore) WriteBlock(piece int, begin int, data []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if piece < 0 || piece >= s.numPieces {
		return false, errors.New("piece out of range")
	}
	if begin < 0 {
		return false, errors.New("begin out of range")
	}
	psize := s.pieceSize(piece)
	if int64(begin) > psize {
		return false, errors.New("begin beyond piece size")
	}
	if int64(begin)+int64(len(data)) > psize {
		return false, errors.New("block exceeds piece size")
	}

	global := int64(piece)*int64(s.pieceLength) + int64(begin)
	if _, err := s.f.WriteAt(data, global); err != nil {
		return false, err
	}

	if !s.completed[piece] {
		s.received[piece] += int64(len(data))
		if s.received[piece] >= psize {
			s.markComplete(piece)
			_ = s.f.Sync()
			// fire callbacks out of lock
			cbs := append([]func(int){}, s.cbs...)
			go func(idx int, list []func(int)) {
				for _, cb := range list {
					cb(idx)
				}
			}(piece, cbs)
			return true, nil
		}
	}
	return false, nil
}

func (s *DiskPieceStore) ReadBlock(piece int, begin int, length int) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if piece < 0 || piece >= s.numPieces {
		return nil, errors.New("piece out of range")
	}
	if begin < 0 || length < 0 {
		return nil, errors.New("invalid range")
	}
	if int64(begin)+int64(length) > s.pieceSize(piece) {
		return nil, errors.New("read beyond piece")
	}
	buf := make([]byte, length)
	global := int64(piece)*int64(s.pieceLength) + int64(begin)
	_, err := s.f.ReadAt(buf, global)
	return buf, err
}

func (s *DiskPieceStore) OnPieceComplete(cb func(piece int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cbs = append(s.cbs, cb)
}
