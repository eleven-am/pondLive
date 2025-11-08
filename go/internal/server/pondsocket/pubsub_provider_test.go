package pondsocket

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
)

type stubPubsubSession struct {
	id runtime.SessionID

	mu         sync.Mutex
	deliveries []struct {
		topic   string
		payload []byte
		meta    map[string]string
	}
	flushErr   error
	flushCalls int
}

func (s *stubPubsubSession) DeliverPubsub(topic string, payload []byte, meta map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payloadCopy := append([]byte(nil), payload...)
	var metaCopy map[string]string
	if meta != nil {
		metaCopy = make(map[string]string, len(meta))
		for k, v := range meta {
			metaCopy[k] = v
		}
	}
	s.deliveries = append(s.deliveries, struct {
		topic   string
		payload []byte
		meta    map[string]string
	}{topic: topic, payload: payloadCopy, meta: metaCopy})
}

func (s *stubPubsubSession) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushCalls++
	return s.flushErr
}

func (s *stubPubsubSession) ID() runtime.SessionID { return s.id }

type stubServerTransport struct {
	mu     sync.Mutex
	errors []protocol.ServerError
}

func (s *stubServerTransport) Close() error                                   { return nil }
func (s *stubServerTransport) SendInit(protocol.Init) error                   { return nil }
func (s *stubServerTransport) SendResume(protocol.ResumeOK) error             { return nil }
func (s *stubServerTransport) SendFrame(protocol.Frame) error                 { return nil }
func (s *stubServerTransport) SendPubsubControl(protocol.PubsubControl) error { return nil }
func (s *stubServerTransport) SendUploadControl(protocol.UploadControl) error { return nil }
func (s *stubServerTransport) SendDOMRequest(protocol.DOMRequest) error       { return nil }
func (s *stubServerTransport) SendEventAck(protocol.EventAck) error           { return nil }
func (s *stubServerTransport) SendServerError(err protocol.ServerError) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
	return nil
}

func TestDeliverToSessionFlushes(t *testing.T) {
	session := &stubPubsubSession{id: runtime.SessionID("sess")}
	transport := &stubServerTransport{}
	provider := &pubsubProvider{}

	envelope := pubsubEnvelope{
		Data: json.RawMessage("\"hello\""),
		Meta: map[string]string{"id": "123"},
	}

	if err := provider.deliverToSession(session, transport, "news", envelope); err != nil {
		t.Fatalf("deliverToSession: %v", err)
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if session.flushCalls != 1 {
		t.Fatalf("expected flush to be called once, got %d", session.flushCalls)
	}
	if len(session.deliveries) != 1 {
		t.Fatalf("expected one delivery, got %d", len(session.deliveries))
	}

	delivery := session.deliveries[0]
	if delivery.topic != "news" {
		t.Fatalf("expected topic news, got %q", delivery.topic)
	}
	if string(delivery.payload) != "\"hello\"" {
		t.Fatalf("unexpected payload %q", string(delivery.payload))
	}
	if delivery.meta["id"] != "123" {
		t.Fatalf("unexpected meta value %q", delivery.meta["id"])
	}

	envelope.Data[0] = 'X'
	envelope.Meta["id"] = "999"
	if string(delivery.payload) != "\"hello\"" {
		t.Fatalf("payload mutated after delivery: %q", string(delivery.payload))
	}
	if delivery.meta["id"] != "123" {
		t.Fatalf("meta mutated after delivery: %q", delivery.meta["id"])
	}

	transport.mu.Lock()
	defer transport.mu.Unlock()
	if len(transport.errors) != 0 {
		t.Fatalf("expected no server errors, got %d", len(transport.errors))
	}
}

func TestDeliverToSessionFlushErrorSendsServerError(t *testing.T) {
	flushErr := errors.New("boom")
	session := &stubPubsubSession{id: runtime.SessionID("sess"), flushErr: flushErr}
	transport := &stubServerTransport{}
	provider := &pubsubProvider{}

	envelope := pubsubEnvelope{Data: json.RawMessage("null")}

	err := provider.deliverToSession(session, transport, "news", envelope)
	if !errors.Is(err, flushErr) {
		t.Fatalf("expected error %v, got %v", flushErr, err)
	}

	session.mu.Lock()
	if session.flushCalls != 1 {
		t.Fatalf("expected flush to be called once, got %d", session.flushCalls)
	}
	session.mu.Unlock()

	transport.mu.Lock()
	defer transport.mu.Unlock()
	if len(transport.errors) != 1 {
		t.Fatalf("expected one server error, got %d", len(transport.errors))
	}
	if transport.errors[0].Code != "flush_failed" {
		t.Fatalf("expected error code flush_failed, got %q", transport.errors[0].Code)
	}
	if transport.errors[0].SID != string(session.id) {
		t.Fatalf("expected server error SID %q, got %q", session.id, transport.errors[0].SID)
	}
}

func TestProcessOutgoingDeliversForCurrentConnection(t *testing.T) {
	registry := server.NewSessionRegistry()
	provider := &pubsubProvider{registry: registry}
	provider.deliver = func(session pubsubSession, transport server.Transport, topic string, envelope pubsubEnvelope) error {
		if session == nil {
			t.Fatal("missing session")
		}
		if topic != "news" {
			t.Fatalf("unexpected topic %q", topic)
		}
		return nil
	}

	component := func(ctx runtime.Ctx, _ struct{}) h.Node { return h.Div() }
	session := runtime.NewLiveSession(runtime.SessionID("sess"), 1, component, struct{}{}, nil)
	registry.Put(session)

	transport := &stubServerTransport{}
	if _, err := registry.Attach(session.ID(), "conn", transport); err != nil {
		t.Fatalf("registry.Attach: %v", err)
	}

	envelope := pubsubEnvelope{Data: json.RawMessage("null")}

	if err := provider.processOutgoingMessage(session.ID(), "conn", "news", envelope); err != nil {
		t.Fatalf("processOutgoingMessage returned error: %v", err)
	}
}

func TestProcessOutgoingSkipsUnknownSession(t *testing.T) {
	registry := server.NewSessionRegistry()
	provider := &pubsubProvider{registry: registry, deliver: func(pubsubSession, server.Transport, string, pubsubEnvelope) error {
		t.Fatal("unexpected delivery")
		return nil
	}}

	envelope := pubsubEnvelope{Data: json.RawMessage("null")}
	if err := provider.processOutgoingMessage(runtime.SessionID("missing"), "conn", "news", envelope); err != nil {
		t.Fatalf("processOutgoingMessage returned error: %v", err)
	}
}

func TestProcessOutgoingSkipsStaleConnection(t *testing.T) {
	registry := server.NewSessionRegistry()
	invoked := false
	provider := &pubsubProvider{registry: registry}
	provider.deliver = func(pubsubSession, server.Transport, string, pubsubEnvelope) error {
		invoked = true
		return nil
	}

	component := func(ctx runtime.Ctx, _ struct{}) h.Node { return h.Div() }
	active := runtime.NewLiveSession(runtime.SessionID("active"), 1, component, struct{}{}, nil)
	stale := runtime.NewLiveSession(runtime.SessionID("stale"), 1, component, struct{}{}, nil)
	registry.Put(active)
	registry.Put(stale)

	transport := &stubServerTransport{}
	if _, err := registry.Attach(active.ID(), "conn", transport); err != nil {
		t.Fatalf("registry.Attach active: %v", err)
	}

	envelope := pubsubEnvelope{Data: json.RawMessage("null")}

	if err := provider.processOutgoingMessage(stale.ID(), "conn", "news", envelope); err != nil {
		t.Fatalf("processOutgoingMessage returned error: %v", err)
	}
	if invoked {
		t.Fatal("expected delivery to be skipped for stale session")
	}
}
