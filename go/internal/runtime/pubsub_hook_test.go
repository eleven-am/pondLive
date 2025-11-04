package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	h "github.com/eleven-am/liveui/pkg/liveui/html"
)

type pubsubHandleCollector struct {
	mu     sync.Mutex
	handle PubsubHandle[string]
}

func (c *pubsubHandleCollector) set(h PubsubHandle[string]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handle = h
}

func (c *pubsubHandleCollector) get() PubsubHandle[string] {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.handle
}

func pubsubTestComponent(ctx Ctx, props *pubsubHandleCollector) h.Node {
	handle := UsePubsub[string](ctx, "chat")
	if props != nil {
		props.set(handle)
	}
	return h.Div()
}

func TestUsePubsubReceivesMessages(t *testing.T) {
	collector := &pubsubHandleCollector{}
	var captured PubsubHandler

	provider := &stubPubsubProvider{
		subscribeFn: func(_ *LiveSession, _ string, handler PubsubHandler) (string, error) {
			captured = handler
			return "tok", nil
		},
		unsubscribeFn: func(*LiveSession, string) error { return nil },
	}

	sess := NewLiveSession("sess", 1, pubsubTestComponent, collector, &LiveSessionConfig{PubsubProvider: provider})
	sess.ComponentSession().InitialStructured()
	sess.MarkDirty()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if captured == nil {
		t.Fatalf("expected subscribe handler")
	}

	captured("chat", []byte(`"hello"`), map[string]string{"k": "v"})
	sess.MarkDirty()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after deliver: %v", err)
	}

	handle := collector.get()
	if !handle.Connected() {
		t.Fatalf("expected connected state")
	}

	latest, ok := handle.Latest()
	if !ok {
		t.Fatalf("expected latest message")
	}
	if latest.Payload != "hello" {
		t.Fatalf("unexpected payload %q", latest.Payload)
	}
	if latest.Meta["k"] != "v" {
		t.Fatalf("metadata not preserved")
	}
	if latest.ReceivedAt.IsZero() || time.Since(latest.ReceivedAt) > time.Second {
		t.Fatalf("unexpected timestamp: %v", latest.ReceivedAt)
	}

	msgs := handle.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
}

func TestUsePubsubPublish(t *testing.T) {
	collector := &pubsubHandleCollector{}
	var recorded struct {
		mu      sync.Mutex
		topic   string
		payload []byte
	}

	provider := &stubPubsubProvider{
		subscribeFn:   func(*LiveSession, string, PubsubHandler) (string, error) { return "tok", nil },
		unsubscribeFn: func(*LiveSession, string) error { return nil },
		publishFn: func(topic string, payload []byte, _ map[string]string) error {
			recorded.mu.Lock()
			recorded.topic = topic
			recorded.payload = append([]byte(nil), payload...)
			recorded.mu.Unlock()
			return nil
		},
	}

	sess := NewLiveSession("sess", 1, pubsubTestComponent, collector, &LiveSessionConfig{PubsubProvider: provider})
	sess.ComponentSession().InitialStructured()
	sess.MarkDirty()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	handle := collector.get()
	if err := handle.Publish(context.Background(), "goodbye"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	recorded.mu.Lock()
	defer recorded.mu.Unlock()
	if recorded.topic != "chat" {
		t.Fatalf("expected topic chat, got %q", recorded.topic)
	}
	if string(recorded.payload) != "\"goodbye\"" {
		t.Fatalf("expected JSON encoded payload, got %s", recorded.payload)
	}
}
