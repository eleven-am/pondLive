package pondsocket

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/session"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type stubTransport struct {
	mu     sync.Mutex
	closed bool
	errors []protocol.ServerError
}

func (s *stubTransport) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *stubTransport) SendInit(protocol.Init) error             { return nil }
func (s *stubTransport) SendResume(protocol.ResumeOK) error       { return nil }
func (s *stubTransport) SendFrame(protocol.Frame) error           { return nil }
func (s *stubTransport) SendEventAck(protocol.EventAck) error     { return nil }
func (s *stubTransport) SendDiagnostic(protocol.Diagnostic) error { return nil }
func (s *stubTransport) SendDOMRequest(protocol.DOMRequest) error { return nil }

func (s *stubTransport) SendServerError(err protocol.ServerError) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
	return nil
}

func newTestSession(id string) *session.LiveSession {
	component := func(ctx runtime.Ctx, _ struct{}) h.Node {
		return h.Div()
	}

	return session.NewLiveSession(session.SessionID(id), 1, component, struct{}{}, &session.Config{
		Clock: time.Now,
		TTL:   time.Minute,
	})
}

func TestPubsubProviderNil(t *testing.T) {
	var p *pubsubProvider

	if _, err := p.Subscribe(context.Background(), "test", nil); err != runtime.ErrPubsubUnavailable {
		t.Fatalf("expected ErrPubsubUnavailable for nil provider, got %v", err)
	}

	if err := p.Unsubscribe(context.Background(), "token"); err != runtime.ErrPubsubUnavailable {
		t.Fatalf("expected ErrPubsubUnavailable for nil provider, got %v", err)
	}

	if err := p.Publish(context.Background(), "test", nil, nil); err != runtime.ErrPubsubUnavailable {
		t.Fatalf("expected ErrPubsubUnavailable for nil provider, got %v", err)
	}
}

func TestPubsubProviderNilHandler(t *testing.T) {
	provider := &pubsubProvider{
		registry:      server.NewSessionRegistry(),
		subscriptions: make(map[string]pubsubSubscription),
		sessionTopics: make(map[session.SessionID]map[string]int),
	}
	sess := newTestSession("handler")
	adapter := WrapSessionPubsubProvider(sess, provider)

	if _, err := adapter.Subscribe(context.Background(), "news", nil); err == nil {
		t.Fatal("expected error for nil handler")
	}
}

func TestSessionPubsubAdapterSubscribeUnsubscribe(t *testing.T) {
	provider := &pubsubProvider{
		registry:      server.NewSessionRegistry(),
		subscriptions: make(map[string]pubsubSubscription),
		sessionTopics: make(map[session.SessionID]map[string]int),
	}

	sess := newTestSession("subsess")
	adapter := WrapSessionPubsubProvider(sess, provider)
	token, err := adapter.Subscribe(context.Background(), "news", func([]byte, map[string]string) {})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	provider.mu.Lock()
	if _, ok := provider.subscriptions[token]; !ok {
		provider.mu.Unlock()
		t.Fatalf("expected subscription recorded")
	}
	provider.mu.Unlock()

	if err := adapter.Unsubscribe(context.Background(), token); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}

	provider.mu.Lock()
	if _, ok := provider.subscriptions[token]; ok {
		provider.mu.Unlock()
		t.Fatalf("expected subscription removed")
	}
	provider.mu.Unlock()
}

func TestCopyStringMap(t *testing.T) {
	src := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	dst := copyStringMap(src)
	if len(dst) != len(src) {
		t.Fatalf("expected copy to have same length, got %d", len(dst))
	}

	for k, v := range src {
		if dst[k] != v {
			t.Fatalf("expected dst[%q] = %q, got %q", k, v, dst[k])
		}
	}

	dst["key1"] = "modified"
	if src["key1"] == "modified" {
		t.Fatal("modification of copy affected original")
	}

	if copyStringMap(nil) != nil {
		t.Fatal("expected nil for nil input")
	}

	if copyStringMap(map[string]string{}) != nil {
		t.Fatal("expected nil for empty map")
	}
}

func TestServerError(t *testing.T) {
	err := serverError(session.SessionID("test-sid"), "test_code", nil)
	if err.T != "error" {
		t.Fatalf("expected T=error, got %q", err.T)
	}
	if err.SID != "test-sid" {
		t.Fatalf("expected SID=test-sid, got %q", err.SID)
	}
	if err.Code != "test_code" {
		t.Fatalf("expected Code=test_code, got %q", err.Code)
	}
	if err.Message != "" {
		t.Fatalf("expected empty message for nil error, got %q", err.Message)
	}

	err = serverError(session.SessionID("test-sid"), "test_code", runtime.ErrPubsubUnavailable)
	if err.Message != runtime.ErrPubsubUnavailable.Error() {
		t.Fatalf("expected error message, got %q", err.Message)
	}
}

func TestPayloadToDOMEvent(t *testing.T) {
	payload := map[string]interface{}{
		"name":  "click",
		"value": "button",
		"form": map[string]interface{}{
			"username": "test",
			"email":    "test@example.com",
		},
		"mods": map[string]interface{}{
			"ctrl":  true,
			"shift": false,
			"alt":   false,
			"meta":  true,
		},
		"extra": "data",
	}

	event := payloadToDOMEvent(payload)

	if event.Name != "click" {
		t.Fatalf("expected Name=click, got %q", event.Name)
	}
	if event.Value != "button" {
		t.Fatalf("expected Value=button, got %q", event.Value)
	}

	if event.Form["username"] != "test" {
		t.Fatalf("expected Form[username]=test, got %q", event.Form["username"])
	}
	if event.Form["email"] != "test@example.com" {
		t.Fatalf("expected Form[email]=test@example.com, got %q", event.Form["email"])
	}

	if !event.Mods.Ctrl {
		t.Fatal("expected Ctrl=true")
	}
	if event.Mods.Shift {
		t.Fatal("expected Shift=false")
	}
	if !event.Mods.Meta {
		t.Fatal("expected Meta=true")
	}

	if event.Payload["extra"] != "data" {
		t.Fatalf("expected Payload[extra]=data, got %v", event.Payload["extra"])
	}

	if _, exists := event.Payload["name"]; exists {
		t.Fatal("standard field 'name' should not be in Payload")
	}
	if _, exists := event.Payload["form"]; exists {
		t.Fatal("standard field 'form' should not be in Payload")
	}
}

func TestPayloadToDOMEventTypeAlias(t *testing.T) {

	payload := map[string]interface{}{
		"type":  "submit",
		"value": "form-submit",
	}

	event := payloadToDOMEvent(payload)

	if event.Name != "submit" {
		t.Fatalf("expected Name=submit from type field, got %q", event.Name)
	}
}

func TestPayloadToDOMEventButton(t *testing.T) {

	payload := map[string]interface{}{
		"name": "mousedown",
		"mods": map[string]interface{}{
			"button": 1,
		},
	}

	event := payloadToDOMEvent(payload)
	if event.Mods.Button != 1 {
		t.Fatalf("expected Button=1, got %d", event.Mods.Button)
	}

	payload = map[string]interface{}{
		"name": "mousedown",
		"mods": map[string]interface{}{
			"button": float64(2),
		},
	}

	event = payloadToDOMEvent(payload)
	if event.Mods.Button != 2 {
		t.Fatalf("expected Button=2, got %d", event.Mods.Button)
	}
}

func TestPubsubEnvelopeMarshaling(t *testing.T) {
	envelope := pubsubEnvelope{
		Data: json.RawMessage(`{"key":"value"}`),
		Meta: map[string]string{
			"from": "test",
		},
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded pubsubEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if string(decoded.Data) != `{"key":"value"}` {
		t.Fatalf("expected data preserved, got %s", decoded.Data)
	}
	if decoded.Meta["from"] != "test" {
		t.Fatalf("expected meta preserved, got %v", decoded.Meta)
	}
}
