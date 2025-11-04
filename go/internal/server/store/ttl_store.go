package store

import (
	"sync"
	"time"

	runtime "github.com/eleven-am/liveui/internal/runtime"
)

// TTLStore persists session expiry metadata.
type TTLStore interface {
	Touch(id runtime.SessionID, ttl time.Duration) error
	Remove(id runtime.SessionID) error
	Expired(now time.Time) ([]runtime.SessionID, error)
}

// NewInMemoryTTLStore returns a TTL store backed by an in-memory map.
func NewInMemoryTTLStore() TTLStore {
	return &inMemoryTTLStore{items: make(map[runtime.SessionID]time.Time)}
}

type inMemoryTTLStore struct {
	mu    sync.Mutex
	items map[runtime.SessionID]time.Time
}

func (s *inMemoryTTLStore) Touch(id runtime.SessionID, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl <= 0 {
		delete(s.items, id)
		return nil
	}
	s.items[id] = time.Now().Add(ttl)
	return nil
}

func (s *inMemoryTTLStore) Remove(id runtime.SessionID) error {
	s.mu.Lock()
	delete(s.items, id)
	s.mu.Unlock()
	return nil
}

func (s *inMemoryTTLStore) Expired(now time.Time) ([]runtime.SessionID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var expired []runtime.SessionID
	for id, exp := range s.items {
		if exp.IsZero() {
			continue
		}
		if !exp.After(now) {
			expired = append(expired, id)
			delete(s.items, id)
		}
	}
	return expired, nil
}
