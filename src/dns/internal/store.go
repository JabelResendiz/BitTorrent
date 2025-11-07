
// internal/store.go
package internal


import (
	"sync"
	"time"
)

var storelog = NewLogger("STORE")

// Store holds the local DNS records and provides tthread-safe access
type Store struct {
	mu sync.RWMutex
	records map[string]Record
}

// New creates and returns a new empty Store
func New() *Store{
	return &Store{records:make(map[string]Record)}
}

// Add inserts or updates a DNS record, refreshing its timestap
func (s *Store) Add(r Record){
	s.mu.Lock()
	defer s.mu.Unlock()

	r.Timestamp = time.Now()
	s.records[r.Name]= r
	storelog.Info("Added/Updated record: %s -> %s (TTL %d)", r.Name, r.IP, r.TTL)
}


func (s *Store) Delete(name string){
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[name]; ok {
        delete(s.records, name)
        storelog.Info("Deleted record: %s", name)
    } else {
        storelog.Warn("Attempted to delete non-existent record: %s", name)
    }
}


func(s* Store) Get(name string) (Record, bool){
	s.mu.RLock()
	defer s.mu.RUnlock()
	r,ok := s.records[name]

	if ok {
        storelog.Debug("Retrieved record: %s -> %s", r.Name, r.IP)
    } else {
        storelog.Debug("Record not found: %s", name)
    }

	return r,ok
}


func (s *Store) List()[]Record{
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := []Record{}
	for _,r := range s.records{
		out = append(out,r)
	}
	
	storelog.Debug("Listing all records, total: %d", len(out))
	return out
}


