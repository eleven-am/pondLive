package store

import (
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/session"
)

func TestNewInMemoryTTLStore(t *testing.T) {
	store := NewInMemoryTTLStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestInMemoryTTLStore_Touch(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := session.SessionID("test-session")

	err := store.Touch(id, time.Hour)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInMemoryTTLStore_Touch_ZeroTTL(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := session.SessionID("test-session")

	err := store.Touch(id, time.Hour)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = store.Touch(id, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expired, _ := store.Expired(time.Now().Add(2 * time.Hour))
	for _, expID := range expired {
		if expID == id {
			t.Error("session should have been removed with zero TTL")
		}
	}
}

func TestInMemoryTTLStore_Touch_NegativeTTL(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := session.SessionID("test-session")

	err := store.Touch(id, time.Hour)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = store.Touch(id, -time.Hour)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInMemoryTTLStore_Remove(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := session.SessionID("test-session")

	_ = store.Touch(id, time.Hour)

	err := store.Remove(id)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expired, _ := store.Expired(time.Now().Add(2 * time.Hour))
	for _, expID := range expired {
		if expID == id {
			t.Error("removed session should not appear in expired list")
		}
	}
}

func TestInMemoryTTLStore_Remove_NonExistent(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := session.SessionID("nonexistent")

	err := store.Remove(id)
	if err != nil {
		t.Errorf("unexpected error removing non-existent: %v", err)
	}
}

func TestInMemoryTTLStore_Expired(t *testing.T) {
	store := NewInMemoryTTLStore()

	id1 := session.SessionID("session-1")
	id2 := session.SessionID("session-2")
	id3 := session.SessionID("session-3")

	now := time.Now()
	_ = store.Touch(id1, -time.Hour)
	_ = store.Touch(id2, time.Hour)
	_ = store.Touch(id3, -time.Minute)

	expired, err := store.Expired(now)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(expired) != 0 {
		t.Errorf("expected 0 expired sessions (negative TTL removes), got %d", len(expired))
	}

	_ = store.Touch(id1, time.Millisecond)
	_ = store.Touch(id3, time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	expired, err = store.Expired(time.Now())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(expired) != 2 {
		t.Errorf("expected 2 expired sessions, got %d", len(expired))
	}
}

func TestInMemoryTTLStore_Expired_Empty(t *testing.T) {
	store := NewInMemoryTTLStore()

	expired, err := store.Expired(time.Now())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(expired) != 0 {
		t.Errorf("expected 0 expired, got %d", len(expired))
	}
}

func TestInMemoryTTLStore_Expired_RemovesFromStore(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := session.SessionID("test-session")

	_ = store.Touch(id, time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	expired1, _ := store.Expired(time.Now())
	if len(expired1) != 1 {
		t.Errorf("expected 1 expired, got %d", len(expired1))
	}

	expired2, _ := store.Expired(time.Now())
	if len(expired2) != 0 {
		t.Errorf("expected 0 expired after already retrieved, got %d", len(expired2))
	}
}

func TestInMemoryTTLStore_Concurrent(t *testing.T) {
	store := NewInMemoryTTLStore()

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			store.Touch(session.SessionID("session-touch"), time.Hour)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			store.Remove(session.SessionID("session-remove"))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			store.Expired(time.Now())
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}
