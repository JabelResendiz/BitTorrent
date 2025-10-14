package peerwire

import (
	"crypto/sha1"
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

	// expected SHA-1 per piece; if provided, completion requires hash match
	expected [][20]byte

	cbs []func(int)
}

func NewDiskPieceStore(path string, pieceLength int, totalLength int64) (*DiskPieceStore, error) {
	return NewDiskPieceStoreWithMode(path, pieceLength, totalLength, true)
}

// NewDiskPieceStoreWithMode allows controlling whether to truncate the file.
// If truncate is true, the file will be created/truncated to totalLength (download mode).
// If false, it will open without truncation (useful for seeding from an existing file).
func NewDiskPieceStoreWithMode(path string, pieceLength int, totalLength int64, truncate bool) (*DiskPieceStore, error) {
	if pieceLength <= 0 || totalLength <= 0 {
		return nil, errors.New("invalid lengths")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if truncate {
		if err := f.Truncate(totalLength); err != nil {
			f.Close()
			return nil, err
		}
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
			// If we have expected hashes, verify before marking complete
			if len(s.expected) == s.numPieces {
				// Read full piece from disk
				plen := int(psize)
				buf := make([]byte, plen)
				off := int64(piece) * int64(s.pieceLength)
				if _, err := s.f.ReadAt(buf, off); err != nil {
					return false, err
				}
				sum := sha1.Sum(buf)
				if sum != s.expected[piece] {
					// Hash mismatch: treat as invalid; reset counters for this piece
					s.received[piece] = 0
					return false, errors.New("piece hash mismatch")
				}
			}
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

// SetExpectedHashes sets the expected SHA-1 hash per piece (20 bytes each).
// Provide exactly NumPieces() entries. If set, a piece will only be marked
// complete once its on-disk bytes hash to the expected value.
func (s *DiskPieceStore) SetExpectedHashes(hashes [][20]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expected = hashes
}

// ScanAndMarkComplete scans the underlying file and, when expected hashes are set,
// marks pieces as complete if their content matches the expected SHA-1.
// It sets received counters accordingly. Intended for seeding from an existing file.
func (s *DiskPieceStore) ScanAndMarkComplete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.expected) != s.numPieces {
		return errors.New("expected hashes not set or length mismatch")
	}
	fi, err := s.f.Stat()
	if err != nil {
		return err
	}
	if fi.Size() != s.totalLength {
		return errors.New("file size does not match total length")
	}
	// Iterate pieces
	for i := 0; i < s.numPieces; i++ {
		if s.completed[i] {
			continue
		}
		psize := s.pieceSize(i)
		buf := make([]byte, int(psize))
		off := int64(i) * int64(s.pieceLength)
		if _, err := s.f.ReadAt(buf, off); err != nil {
			return err
		}
		sum := sha1.Sum(buf)
		if sum == s.expected[i] {
			// mark complete
			byteIdx := i / 8
			bit := 7 - (i % 8)
			s.bitfield[byteIdx] |= (1 << uint(bit))
			s.completed[i] = true
			s.received[i] = psize
		} else {
			s.completed[i] = false
			s.received[i] = 0
		}
	}
	return nil
}
