package runtime

import (
	"sync"
	"testing"

	h "github.com/eleven-am/liveui/pkg/liveui/html"
)

type stubPubsubProvider struct {
	subscribeFn   func(*LiveSession, string, PubsubHandler) (string, error)
	unsubscribeFn func(*LiveSession, string) error
	publishFn     func(string, []byte, map[string]string) error
}

func (s *stubPubsubProvider) Subscribe(session *LiveSession, topic string, handler PubsubHandler) (string, error) {
	if s.subscribeFn != nil {
		return s.subscribeFn(session, topic, handler)
	}
	return "", nil
}

func (s *stubPubsubProvider) Unsubscribe(session *LiveSession, token string) error {
	if s.unsubscribeFn != nil {
		return s.unsubscribeFn(session, token)
	}
	return nil
}

func (s *stubPubsubProvider) Publish(topic string, payload []byte, meta map[string]string) error {
	if s.publishFn != nil {
		return s.publishFn(topic, payload, meta)
	}
	return nil
}

func TestComponentSessionSubscribeWithoutProvider(t *testing.T) {
	session := NewLiveSession(SessionID("sess"), 1, func(Ctx, struct{}) h.Node {
		return h.Fragment()
	}, struct{}{}, nil)

	if _, err := session.ComponentSession().subscribePubsub("news", nil, func([]byte, map[string]string) {}); err != ErrPubsubUnavailable {
		t.Fatalf("expected ErrPubsubUnavailable, got %v", err)
	}
}

func TestComponentSessionDeliverPubsubQueuesHandlers(t *testing.T) {
	var (
		handlerLock sync.Mutex
		received    []string
	)

	var captured PubsubHandler
	provider := &stubPubsubProvider{
		subscribeFn: func(_ *LiveSession, topic string, handler PubsubHandler) (string, error) {
			if topic != "news" {
				t.Fatalf("unexpected topic %q", topic)
			}
			captured = handler
			return "token", nil
		},
	}

	session := NewLiveSession(SessionID("sess"), 1, func(Ctx, struct{}) h.Node {
		return h.Fragment()
	}, struct{}{}, &LiveSessionConfig{PubsubProvider: provider})

	comp := session.ComponentSession()
	if comp == nil {
		t.Fatal("expected component session")
	}

	token, err := comp.subscribePubsub("news", nil, func(payload []byte, meta map[string]string) {
		handlerLock.Lock()
		defer handlerLock.Unlock()
		received = append(received, string(payload)+":"+meta["id"])
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
	if captured == nil {
		t.Fatalf("provider did not capture handler")
	}

	captured("news", []byte("hello"), map[string]string{"id": "1"})
	captured("other", []byte("ignore"), map[string]string{"id": "2"})

	if err := comp.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	handlerLock.Lock()
	defer handlerLock.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(received))
	}
	if received[0] != "hello:1" {
		t.Fatalf("unexpected payload %q", received[0])
	}
}

func TestComponentSessionPublishDelegates(t *testing.T) {
	var (
		gotTopic   string
		gotPayload []byte
		gotMeta    map[string]string
	)

	provider := &stubPubsubProvider{
		publishFn: func(topic string, payload []byte, meta map[string]string) error {
			gotTopic = topic
			gotPayload = payload
			gotMeta = meta
			return nil
		},
	}

	session := NewLiveSession(SessionID("sess"), 1, func(Ctx, struct{}) h.Node {
		return h.Fragment()
	}, struct{}{}, &LiveSessionConfig{PubsubProvider: provider})

	comp := session.ComponentSession()
	payload := []byte("hello")
	meta := map[string]string{"id": "123"}

	if err := comp.publishPubsub("news", nil, payload, meta); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if gotTopic != "news" {
		t.Fatalf("expected topic news, got %q", gotTopic)
	}
	if string(gotPayload) != "hello" {
		t.Fatalf("expected payload copy \"hello\", got %q", string(gotPayload))
	}
	if gotMeta == nil || gotMeta["id"] != "123" {
		t.Fatalf("expected meta copy, got %#v", gotMeta)
	}

	payload[0] = 'X'
	meta["id"] = "456"

	if string(gotPayload) != "hello" {
		t.Fatalf("payload mutated after publish: %q", string(gotPayload))
	}
	if gotMeta["id"] != "123" {
		t.Fatalf("meta mutated after publish: %q", gotMeta["id"])
	}
}

func TestComponentSessionUnsubscribeDelegates(t *testing.T) {
	var (
		captured PubsubHandler
		unsubTok string
	)

	provider := &stubPubsubProvider{
		subscribeFn: func(_ *LiveSession, topic string, handler PubsubHandler) (string, error) {
			captured = handler
			return "token", nil
		},
		unsubscribeFn: func(_ *LiveSession, token string) error {
			unsubTok = token
			return nil
		},
	}

	session := NewLiveSession(SessionID("sess"), 1, func(Ctx, struct{}) h.Node {
		return h.Fragment()
	}, struct{}{}, &LiveSessionConfig{PubsubProvider: provider})

	comp := session.ComponentSession()
	token, err := comp.subscribePubsub("news", nil, func([]byte, map[string]string) {})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if captured == nil {
		t.Fatalf("expected handler capture")
	}

	if err := comp.unsubscribePubsub(token); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if unsubTok != token {
		t.Fatalf("expected unsubscribe token %q, got %q", token, unsubTok)
	}
}
