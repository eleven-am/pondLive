package server

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

func dummyComponent(_ *runtime.Ctx) work.Node {
	return &work.Element{Tag: "div"}
}

func newSess(t *testing.T, id string, onClose func(*session.LiveSession, session.CloseReason)) *session.LiveSession {
	t.Helper()
	s := session.NewLiveSession(session.SessionID(id), 1, dummyComponent, &session.Config{OnClose: onClose})
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func waitDone(t *testing.T, s *session.LiveSession) {
	t.Helper()
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session did not close")
	}
}

func TestRegistryBasicOperations(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	sess := newSess(t, "test-session", reg.sessionClosed)

	if err := reg.Put(sess); err != nil {
		t.Fatal(err)
	}
	found, ok := reg.Lookup("test-session")
	if !ok || found != sess {
		t.Fatal("expected the same session instance")
	}
	if _, ok = reg.Lookup("nonexistent"); ok {
		t.Error("expected nonexistent session to not be found")
	}
	if err := reg.Put(nil); err == nil {
		t.Error("Put(nil) must fail")
	}

	reg.Remove(sess)
	waitDone(t, sess)
	if _, ok = reg.Lookup("test-session"); ok {
		t.Error("expected session to be removed")
	}
	if sess.CloseReason() != session.ReasonRemoved {
		t.Errorf("close reason = %v, want removed", sess.CloseReason())
	}
}

func TestRegistryDuplicateIDIsAnExplicitErrorAndKeepsTheOldSession(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	first := newSess(t, "same", reg.sessionClosed)
	second := newSess(t, "same", reg.sessionClosed)

	if err := reg.Put(first); err != nil {
		t.Fatal(err)
	}
	if err := reg.Put(second); !errors.Is(err, ErrDuplicateSession) {
		t.Fatalf("second Put returned %v, want ErrDuplicateSession", err)
	}
	if got, _ := reg.Lookup("same"); got != first {
		t.Fatal("the duplicate replaced or paired with the old session")
	}
	_ = second.Close()
	if got, _ := reg.Lookup("same"); got != first {
		t.Fatal("closing the rejected duplicate removed the old session")
	}
	if reg.Stats().Rejected.Duplicate != 1 {
		t.Fatalf("stats = %+v", reg.Stats())
	}
}

func TestRegistryRemoveIsByIdentity(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	first := newSess(t, "same", reg.sessionClosed)
	stale := newSess(t, "same", reg.sessionClosed)
	if err := reg.Put(first); err != nil {
		t.Fatal(err)
	}
	reg.Remove(stale)
	if got, _ := reg.Lookup("same"); got != first {
		t.Fatal("Remove(stale) deleted the live session registered under the same id")
	}
}

func TestRegistryFullRefusesNewSessions(t *testing.T) {
	reg := NewSessionRegistry(2, 2)
	a, b, c := newSess(t, "a", reg.sessionClosed), newSess(t, "b", reg.sessionClosed), newSess(t, "c", reg.sessionClosed)
	if err := reg.Put(a); err != nil {
		t.Fatal(err)
	}
	if err := reg.Put(b); err != nil {
		t.Fatal(err)
	}
	if err := reg.Put(c); !errors.Is(err, ErrRegistryFull) {
		t.Fatalf("Put over capacity returned %v", err)
	}
	reg.Remove(a)
	if err := reg.Put(c); err != nil {
		t.Fatalf("capacity must free up after a removal: %v", err)
	}
	if st := reg.Stats(); st.Capacity != 2 || st.Rejected.Full != 1 {
		t.Fatalf("stats = %+v", st)
	}
}

func TestRegistryClosedRefusesNewSessions(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	reg.Close()
	if err := reg.Put(newSess(t, "a", reg.sessionClosed)); !errors.Is(err, ErrRegistryClosed) {
		t.Fatalf("Put after Close returned %v", err)
	}
}

func TestRegistryStatsCountSessionsPerState(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	pending := newSess(t, "pending", reg.sessionClosed)
	expiring := newSess(t, "expiring", reg.sessionClosed)
	_ = reg.Put(pending)
	_ = reg.Put(expiring)

	if st := reg.Stats(); st.Sessions.Pending != 2 {
		t.Fatalf("stats = %+v", st)
	}
	expiring.RequestClose(session.ReasonPendingExpired)
	waitDone(t, expiring)
	st := reg.Stats()
	if st.Sessions.Pending != 1 || st.Evicted != 1 || st.Closed["pending-expired"] != 1 {
		t.Fatalf("stats after eviction = %+v", st)
	}
}

func TestRegistryRange(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	for _, id := range []string{"sess-1", "sess-2", "sess-3"} {
		_ = reg.Put(newSess(t, id, reg.sessionClosed))
	}

	visited := make(map[session.SessionID]bool)
	reg.Range(func(sess *session.LiveSession) bool {
		visited[sess.ID()] = true
		return true
	})
	if len(visited) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(visited))
	}

	count := 0
	reg.Range(func(*session.LiveSession) bool {
		count++
		return false
	})
	if count != 1 {
		t.Errorf("expected early exit after 1, got %d", count)
	}
}

func TestRegistryConcurrentPutRemoveRange(t *testing.T) {
	reg := NewSessionRegistry(0, 0)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s := session.NewLiveSession("shared", 1, dummyComponent, &session.Config{OnClose: reg.sessionClosed})
				if reg.Put(s) == nil {
					reg.Lookup("shared")
					reg.Remove(s)
				}
				_ = s.Close()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				reg.Range(func(*session.LiveSession) bool { return true })
				_ = reg.Stats()
			}
		}()
	}
	wg.Wait()
	if reg.Len() != 0 {
		t.Fatalf("%d sessions leaked in the registry", reg.Len())
	}
}
