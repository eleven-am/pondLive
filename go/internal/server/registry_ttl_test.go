package server

import (
	"sync"
	"testing"
	"time"

	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type stubTTLStore struct {
	sync.Mutex
	touch   []runtime.SessionID
	remove  []runtime.SessionID
	expired []runtime.SessionID
}

func (s *stubTTLStore) Touch(id runtime.SessionID, ttl time.Duration) error {
	s.Lock()
	s.touch = append(s.touch, id)
	s.Unlock()
	return nil
}

func (s *stubTTLStore) Remove(id runtime.SessionID) error {
	s.Lock()
	s.remove = append(s.remove, id)
	s.Unlock()
	return nil
}

func (s *stubTTLStore) Expired(now time.Time) ([]runtime.SessionID, error) {
	return s.expired, nil
}

func TestRegistryUsesTTLStore(t *testing.T) {
	ttl := &stubTTLStore{}
	reg := NewSessionRegistryWithTTL(ttl)

	sess := runtime.NewLiveSession("sess", 1, func(runtime.Ctx, struct{}) h.Node { return h.Div() }, struct{}{}, nil)
	reg.Put(sess)

	if len(ttl.touch) == 0 || ttl.touch[0] != sess.ID() {
		t.Fatalf("expected ttl touch on Put, got %v", ttl.touch)
	}

	ttl.expired = []runtime.SessionID{sess.ID()}
	expired := reg.SweepExpired()
	if len(expired) != 1 || expired[0] != sess.ID() {
		t.Fatalf("expected expired id, got %v", expired)
	}

	if len(ttl.remove) == 0 || ttl.remove[len(ttl.remove)-1] != sess.ID() {
		t.Fatalf("expected ttl remove when session expired, got %v", ttl.remove)
	}

	reg.Put(sess)
	reg.Remove(sess.ID())
	if len(ttl.remove) == 0 || ttl.remove[len(ttl.remove)-1] != sess.ID() {
		t.Fatalf("expected ttl remove when session deleted, got %v", ttl.remove)
	}
}

func TestNewSessionRegistryDefaultTTL(t *testing.T) {
	reg := NewSessionRegistry()
	sess := runtime.NewLiveSession("sess", 1, func(runtime.Ctx, struct{}) h.Node { return h.Div() }, struct{}{}, nil)
	reg.Put(sess)

	expired := reg.SweepExpired()
	if len(expired) != 0 {
		t.Fatalf("expected no expired sessions by default, got %v", expired)
	}
}

func waitUntil(t *testing.T, fn func() bool) {
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	if !fn() {
		t.Fatalf("condition not met before timeout")
	}
}
