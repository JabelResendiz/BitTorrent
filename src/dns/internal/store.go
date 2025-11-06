

package internal


import (
	"sync"
	"time"
)


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
}


func (s *Store) Delete(name string){
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records,name)
}


func(s* Store) Get(name string) (Record, bool){
	s.mu.RLock()
	defer s.mu.RUnlock()
	r,ok := s.records[name]
	return r,ok
}


func (s *Store) List()[]Record{
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Record{}

	for _,r := range s.records{
		out = append(out,r)
	}

	return out
}


