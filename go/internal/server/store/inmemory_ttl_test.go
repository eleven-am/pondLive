package store

import (
	"testing"
	"time"

	runtime "github.com/eleven-am/liveui/internal/runtime"
)

func TestInMemoryTTLStore(t *testing.T) {
	store := NewInMemoryTTLStore()
	id := runtime.SessionID("sess1")

	if err := store.Touch(id, 10*time.Millisecond); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if expired, _ := store.Expired(time.Now()); len(expired) != 0 {
		t.Fatalf("expected no expirations yet, got %v", expired)
	}
	if err := store.Touch(id, -1); err != nil {
		t.Fatalf("touch negative: %v", err)
	}
	if expired, _ := store.Expired(time.Now().Add(time.Second)); len(expired) != 0 {
		t.Fatalf("expected no expirations after removal, got %v", expired)
	}

	if err := store.Touch(id, 1*time.Millisecond); err != nil {
		t.Fatalf("touch: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	expired, _ := store.Expired(time.Now())
	if len(expired) != 1 || expired[0] != id {
		t.Fatalf("expected id expired, got %v", expired)
	}
}
