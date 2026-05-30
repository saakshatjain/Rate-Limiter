package store
import (
	"time"
	"sync"
)

type MemoryStore struct {
	data map[string]*ClientData
	mu sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]*ClientData),
	}
}

func (s *MemoryStore) Get(key string) (*ClientData, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data[key], nil
}

func (s *MemoryStore) Set(key string , data *ClientData, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = data
	return nil
}

func (s *MemoryStore) Delete(key string) error {	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *MemoryStore) Cleanup(olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	for key, data := range s.data {
		if data.LastRequest.Before(cutoff) {
			delete(s.data, key)
		}
	}
	return nil
}