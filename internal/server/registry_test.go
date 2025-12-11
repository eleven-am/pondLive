package server

import (
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

type mockTransport struct {
	closed bool
	mu     sync.Mutex
}

func (m *mockTransport) Send(topic, event string, data any) error { return nil }
func (m *mockTransport) IsLive() bool                             { return true }
func (m *mockTransport) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}
func (m *mockTransport) RequestInfo() *headers.RequestInfo   { return nil }
func (m *mockTransport) RequestState() *headers.RequestState { return nil }

func dummyComponent(_ *runtime.Ctx) work.Node {
	return &work.Element{Tag: "div"}
}

func TestRegistryBasicOperations(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	found, ok := reg.Lookup("test-session")
	if !ok {
		t.Fatal("expected session to be found")
	}
	if found != sess {
		t.Error("expected same session instance")
	}

	_, ok = reg.Lookup("nonexistent")
	if ok {
		t.Error("expected nonexistent session to not be found")
	}

	reg.Put(nil)

	reg.Remove("test-session")

	_, ok = reg.Lookup("test-session")
	if ok {
		t.Error("expected session to be removed")
	}
}

func TestRegistryAttachDetach(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	transport := &mockTransport{}
	attached, err := reg.Attach("test-session", "conn-1", transport)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if attached != sess {
		t.Error("expected same session")
	}

	foundSess, foundTransport, ok := reg.LookupWithConnection("test-session", "conn-1")
	if !ok {
		t.Fatal("expected to find session with connection")
	}
	if foundSess != sess {
		t.Error("expected same session")
	}
	if foundTransport != transport {
		t.Error("expected same transport")
	}

	foundSess, foundTransport, ok = reg.LookupByConnection("conn-1")
	if !ok {
		t.Fatal("expected to find by connection")
	}
	if foundSess != sess {
		t.Error("expected same session")
	}
	if foundTransport != transport {
		t.Error("expected same transport")
	}

	connID, connTransport, ok := reg.ConnectionForSession("test-session")
	if !ok {
		t.Fatal("expected to find connection for session")
	}
	if connID != "conn-1" {
		t.Errorf("expected conn-1, got %s", connID)
	}
	if connTransport != transport {
		t.Error("expected same transport")
	}

	reg.Detach("conn-1")

	_, _, ok = reg.LookupByConnection("conn-1")
	if ok {
		t.Error("expected connection to be detached")
	}

	transport.mu.Lock()
	closed := transport.closed
	transport.mu.Unlock()
	if !closed {
		t.Error("expected transport to be closed on detach")
	}
}

func TestRegistryAttachReplacesOldTransport(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	transport1 := &mockTransport{}
	_, err := reg.Attach("test-session", "conn-1", transport1)
	if err != nil {
		t.Fatalf("first attach failed: %v", err)
	}

	transport2 := &mockTransport{}
	_, err = reg.Attach("test-session", "conn-2", transport2)
	if err != nil {
		t.Fatalf("second attach failed: %v", err)
	}

	transport1.mu.Lock()
	closed1 := transport1.closed
	transport1.mu.Unlock()
	if !closed1 {
		t.Error("expected old transport to be closed")
	}

	_, _, ok := reg.LookupByConnection("conn-1")
	if ok {
		t.Error("expected old connection to be removed")
	}

	_, foundTransport, ok := reg.LookupByConnection("conn-2")
	if !ok {
		t.Fatal("expected new connection")
	}
	if foundTransport != transport2 {
		t.Error("expected new transport")
	}
}

func TestRegistryAttachSessionNotFound(t *testing.T) {
	reg := NewSessionRegistry()

	transport := &mockTransport{}
	_, err := reg.Attach("nonexistent", "conn-1", transport)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestRegistryLookupWithConnectionWrongConn(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	transport := &mockTransport{}
	_, _ = reg.Attach("test-session", "conn-1", transport)

	foundSess, foundTransport, ok := reg.LookupWithConnection("test-session", "wrong-conn")
	if ok {
		t.Error("expected ok to be false for wrong connection")
	}
	if foundSess != sess {
		t.Error("expected session to still be returned")
	}
	if foundTransport != nil {
		t.Error("expected nil transport for wrong connection")
	}
}

func TestRegistryDetachEmptyConnID(t *testing.T) {
	reg := NewSessionRegistry()
	reg.Detach("")
}

func TestRegistryLookupByConnectionEmptyConnID(t *testing.T) {
	reg := NewSessionRegistry()
	_, _, ok := reg.LookupByConnection("")
	if ok {
		t.Error("expected false for empty connID")
	}
}

func TestLookupWithConnectionRace(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
				}
				transport := &mockTransport{}
				connID := "conn"
				_, _ = reg.Attach("test-session", connID, transport)
				reg.Detach(connID)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
				}
				reg.LookupWithConnection("test-session", "conn")
			}
		}()
	}

	wg.Wait()
	close(done)
}

func TestLookupByConnectionRace(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
				}
				transport := &mockTransport{}
				connID := "conn"
				_, _ = reg.Attach("test-session", connID, transport)
				reg.Detach(connID)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
				}
				reg.LookupByConnection("conn")
			}
		}()
	}

	wg.Wait()
	close(done)
}

func TestConnectionForSessionRace(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)

	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
				}
				transport := &mockTransport{}
				connID := "conn"
				_, _ = reg.Attach("test-session", connID, transport)
				reg.Detach(connID)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
				}
				reg.ConnectionForSession("test-session")
			}
		}()
	}

	wg.Wait()
	close(done)
}

func TestRegistryConcurrentPutLookup(t *testing.T) {
	reg := NewSessionRegistry()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				sess := session.NewLiveSession(session.SessionID("sess"), 1, dummyComponent, nil)
				reg.Put(sess)
				reg.Lookup(session.SessionID("sess"))
				reg.Remove(session.SessionID("sess"))
				sess.Close()
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistryPutIdempotent(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	reg.Put(sess)
	reg.Put(sess)

	found, ok := reg.Lookup("test-session")
	if !ok {
		t.Fatal("expected session")
	}
	if found != sess {
		t.Error("expected same session")
	}
}

func TestRegistryRemoveNonexistent(t *testing.T) {
	reg := NewSessionRegistry()
	reg.Remove("nonexistent")
}

func TestRegistryRange(t *testing.T) {
	reg := NewSessionRegistry()

	sess1 := session.NewLiveSession("sess-1", 1, dummyComponent, nil)
	sess2 := session.NewLiveSession("sess-2", 1, dummyComponent, nil)
	sess3 := session.NewLiveSession("sess-3", 1, dummyComponent, nil)
	defer sess1.Close()
	defer sess2.Close()
	defer sess3.Close()

	reg.Put(sess1)
	reg.Put(sess2)
	reg.Put(sess3)

	visited := make(map[session.SessionID]bool)
	reg.Range(func(sess *session.LiveSession) bool {
		visited[sess.ID()] = true
		return true
	})

	if len(visited) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(visited))
	}
	if !visited["sess-1"] || !visited["sess-2"] || !visited["sess-3"] {
		t.Error("expected all sessions to be visited")
	}
}

func TestRegistryRangeEarlyExit(t *testing.T) {
	reg := NewSessionRegistry()

	sess1 := session.NewLiveSession("sess-1", 1, dummyComponent, nil)
	sess2 := session.NewLiveSession("sess-2", 1, dummyComponent, nil)
	sess3 := session.NewLiveSession("sess-3", 1, dummyComponent, nil)
	defer sess1.Close()
	defer sess2.Close()
	defer sess3.Close()

	reg.Put(sess1)
	reg.Put(sess2)
	reg.Put(sess3)

	count := 0
	reg.Range(func(sess *session.LiveSession) bool {
		count++
		return false
	})

	if count != 1 {
		t.Errorf("expected early exit after 1, got %d", count)
	}
}

func TestRegistryRangeEmpty(t *testing.T) {
	reg := NewSessionRegistry()

	count := 0
	reg.Range(func(sess *session.LiveSession) bool {
		count++
		return true
	})

	if count != 0 {
		t.Errorf("expected 0 sessions, got %d", count)
	}
}

func TestRegistryRangeConcurrent(t *testing.T) {
	reg := NewSessionRegistry()

	sess := session.NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()
	reg.Put(sess)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				reg.Range(func(s *session.LiveSession) bool {
					return true
				})
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s := session.NewLiveSession(session.SessionID("temp"), 1, dummyComponent, nil)
				reg.Put(s)
				reg.Remove(session.SessionID("temp"))
				s.Close()
			}
		}(i)
	}

	wg.Wait()
}
