package store

import (
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/session"
)

type TTLStore interface {
	Touch(id session.SessionID, ttl time.Duration) error
	Remove(id session.SessionID) error
	Expired(now time.Time) ([]session.SessionID, error)
}

func NewInMemoryTTLStore() TTLStore {
	return &inMemoryTTLStore{items: make(map[session.SessionID]time.Time)}
}

type inMemoryTTLStore struct {
	mu    sync.Mutex
	items map[session.SessionID]time.Time
}

func (s *inMemoryTTLStore) Touch(id session.SessionID, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ttl <= 0 {
		delete(s.items, id)
		return nil
	}
	s.items[id] = time.Now().Add(ttl)
	return nil
}

func (s *inMemoryTTLStore) Remove(id session.SessionID) error {
	s.mu.Lock()
	delete(s.items, id)
	s.mu.Unlock()
	return nil
}

func (s *inMemoryTTLStore) Expired(now time.Time) ([]session.SessionID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var expired []session.SessionID
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
