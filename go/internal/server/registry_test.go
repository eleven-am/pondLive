package server

import (
	"sync"
	"testing"
	"time"

	h "github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type stubTransport struct {
	mu       sync.Mutex
	closed   bool
	inits    []protocol.Init
	resumes  []protocol.ResumeOK
	frames   []protocol.Frame
	acks     []protocol.EventAck
	errors   []protocol.ServerError
	controls []protocol.PubsubControl
	uploads  []protocol.UploadControl
	dom      []protocol.DOMRequest
}

func (s *stubTransport) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *stubTransport) SendInit(init protocol.Init) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inits = append(s.inits, init)
	return nil
}

func (s *stubTransport) SendResume(res protocol.ResumeOK) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resumes = append(s.resumes, res)
	return nil
}

func (s *stubTransport) SendFrame(frame protocol.Frame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = append(s.frames, frame)
	return nil
}

func (s *stubTransport) SendEventAck(ack protocol.EventAck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acks = append(s.acks, ack)
	return nil
}

func (s *stubTransport) SendServerError(err protocol.ServerError) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
	return nil
}

func (s *stubTransport) SendPubsubControl(ctrl protocol.PubsubControl) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.controls = append(s.controls, ctrl)
	return nil
}

func (s *stubTransport) SendUploadControl(ctrl protocol.UploadControl) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploads = append(s.uploads, ctrl)
	return nil
}

func (s *stubTransport) SendDOMRequest(req protocol.DOMRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dom = append(s.dom, req)
	return nil
}

func newSession(t *testing.T) (*runtime.LiveSession, func(time.Duration)) {
	return newSessionWithID(t, "sid")
}

func newSessionWithID(t *testing.T, id runtime.SessionID) (*runtime.LiveSession, func(time.Duration)) {
	t.Helper()

	component := func(ctx runtime.Ctx, _ struct{}) h.Node {
		return h.Div()
	}

	current := time.Unix(0, 0)
	clock := func() time.Time { return current }
	advance := func(d time.Duration) { current = current.Add(d) }

	sess := runtime.NewLiveSession(id, 1, component, struct{}{}, &runtime.LiveSessionConfig{
		Clock: clock,
		TTL:   time.Second,
	})

	return sess, advance
}

func TestRegistryAttachLookup(t *testing.T) {
	reg := NewSessionRegistry()
	sess, _ := newSession(t)
	reg.Put(sess)

	got, ok := reg.Lookup(sess.ID())
	if !ok || got != sess {
		t.Fatalf("expected session registered in registry")
	}

	st := &stubTransport{}
	attached, err := reg.Attach(sess.ID(), "conn-1", st)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if attached != sess {
		t.Fatalf("expected same session pointer back from attach")
	}

	if _, _, ok := reg.LookupByConnection("conn-1"); !ok {
		t.Fatalf("expected lookup by connection to succeed")
	}

	reg.DetachConnection("conn-1")
	if _, _, ok := reg.LookupByConnection("conn-1"); ok {
		t.Fatalf("expected connection to be detached")
	}
}

func TestRegistryRemoveClosesTransport(t *testing.T) {
	reg := NewSessionRegistry()
	sess, _ := newSession(t)
	reg.Put(sess)

	st := &stubTransport{}
	if _, err := reg.Attach(sess.ID(), "conn-1", st); err != nil {
		t.Fatalf("attach: %v", err)
	}

	reg.Remove(sess.ID())
	if _, ok := reg.Lookup(sess.ID()); ok {
		t.Fatalf("expected session removed after registry remove")
	}
	if !st.closed {
		t.Fatalf("expected transport to be closed on remove")
	}
}

func TestRegistryAttachEvictsExistingConnection(t *testing.T) {
	reg := NewSessionRegistry()
	first, _ := newSessionWithID(t, "sid-1")
	second, _ := newSessionWithID(t, "sid-2")

	reg.Put(first)
	reg.Put(second)

	firstTransport := &stubTransport{}
	if _, err := reg.Attach(first.ID(), "conn-1", firstTransport); err != nil {
		t.Fatalf("attach first: %v", err)
	}

	if conn, _, ok := reg.ConnectionForSession(first.ID()); !ok || conn != "conn-1" {
		t.Fatalf("expected first session bound to connection, got %q ok=%v", conn, ok)
	}

	secondTransport := &stubTransport{}
	if _, err := reg.Attach(second.ID(), "conn-1", secondTransport); err != nil {
		t.Fatalf("attach second: %v", err)
	}

	if !firstTransport.closed {
		t.Fatalf("expected previous transport to be closed")
	}

	if conn, _, ok := reg.ConnectionForSession(first.ID()); ok {
		t.Fatalf("expected first session detached after eviction, still bound to %q", conn)
	}

	if conn, transport, ok := reg.ConnectionForSession(second.ID()); !ok || conn != "conn-1" || transport != secondTransport {
		t.Fatalf("expected second session to own connection, got conn=%q transport=%v ok=%v", conn, transport, ok)
	}
}

func TestRegistrySweepExpired(t *testing.T) {
	reg := NewSessionRegistry()
	sess, advance := newSession(t)
	reg.Put(sess)

	advance(2 * time.Second)
	expired := reg.SweepExpired()
	if len(expired) != 1 || expired[0] != sess.ID() {
		t.Fatalf("expected sweep to prune expired session, got %v", expired)
	}
}

func TestLookupWithConnection(t *testing.T) {
	reg := NewSessionRegistry()
	sess, _ := newSession(t)
	reg.Put(sess)

	st := &stubTransport{}
	if _, err := reg.Attach(sess.ID(), "conn-1", st); err != nil {
		t.Fatalf("attach: %v", err)
	}

	got, transport, ok := reg.LookupWithConnection(sess.ID(), "conn-1")
	if !ok || got != sess || transport != st {
		t.Fatalf("expected lookup with connection to return session and transport, got sess=%v transport=%v ok=%v", got, transport, ok)
	}

	got, transport, ok = reg.LookupWithConnection(sess.ID(), "other")
	if ok || got != sess || transport != nil {
		t.Fatalf("expected mismatch connection to return session without transport, got sess=%v transport=%v ok=%v", got, transport, ok)
	}

	if _, _, ok := reg.LookupWithConnection(runtime.SessionID("missing"), "conn-1"); ok {
		t.Fatalf("expected missing session lookup to fail")
	}
}
