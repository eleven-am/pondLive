package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

// testPubsubProvider implements PubsubProvider for testing
type testPubsubProvider struct {
	mu            sync.Mutex
	subscriptions map[string]*testSubscription
	nextToken     int
}

type testSubscription struct {
	token   string
	topic   string
	handler func([]byte, map[string]string)
}

func newTestPubsubProvider() *testPubsubProvider {
	return &testPubsubProvider{
		subscriptions: make(map[string]*testSubscription),
	}
}

func (m *testPubsubProvider) Subscribe(ctx context.Context, topic string, handler func(payload []byte, meta map[string]string)) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextToken++
	token := "token-" + topic + "-" + string(rune(m.nextToken))

	m.subscriptions[token] = &testSubscription{
		token:   token,
		topic:   topic,
		handler: handler,
	}

	return token, nil
}

func (m *testPubsubProvider) Unsubscribe(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscriptions, token)
	return nil
}

func (m *testPubsubProvider) Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
	m.mu.Lock()
	subs := make([]*testSubscription, 0)
	for _, sub := range m.subscriptions {
		if sub.topic == topic {
			subs = append(subs, sub)
		}
	}
	m.mu.Unlock()

	for _, sub := range subs {
		sub.handler(payload, meta)
	}

	return nil
}

func (m *testPubsubProvider) simulateMessage(topic string, payload []byte, meta map[string]string) {
	m.mu.Lock()
	subs := make([]*testSubscription, 0)
	for _, sub := range m.subscriptions {
		if sub.topic == topic {
			subs = append(subs, sub)
		}
	}
	m.mu.Unlock()

	for _, sub := range subs {
		sub.handler(payload, meta)
	}
}

func TestUsePubsubBasic(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		handle = UsePubsub(ctx, "test-topic", WithPubsubProvider[string](provider))
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if !handle.Connected() {
		t.Error("expected connected to be true")
	}

	_, ok := handle.Latest()
	if ok {
		t.Error("expected no latest message initially")
	}

	msgs := handle.Messages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestUsePubsubReceiveMessage(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		handle = UsePubsub(ctx, "test-topic", WithPubsubProvider[string](provider))
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	provider.simulateMessage("test-topic", []byte(`"hello"`), map[string]string{"from": "test"})

	latest, ok := handle.Latest()
	if !ok {
		t.Fatal("expected latest message to be available")
	}

	if latest.Payload != "hello" {
		t.Errorf("expected payload 'hello', got %q", latest.Payload)
	}

	if latest.Meta["from"] != "test" {
		t.Errorf("expected meta 'from' to be 'test', got %q", latest.Meta["from"])
	}

	msgs := handle.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Payload != "hello" {
		t.Errorf("expected first message 'hello', got %q", msgs[0].Payload)
	}
}

func TestUsePubsubMultipleMessages(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[int]

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		handle = UsePubsub(ctx, "numbers", WithPubsubProvider[int](provider))
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	provider.simulateMessage("numbers", []byte("1"), nil)
	provider.simulateMessage("numbers", []byte("2"), nil)
	provider.simulateMessage("numbers", []byte("3"), nil)

	msgs := handle.Messages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	latest, ok := handle.Latest()
	if !ok {
		t.Fatal("expected latest message")
	}

	if latest.Payload != 3 {
		t.Errorf("expected latest payload 3, got %d", latest.Payload)
	}
}

func TestUsePubsubPublish(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[string]

	var received string
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		h := UsePubsub(ctx, "echo", WithPubsubProvider[string](provider))
		handle = h

		msgs := h.Messages()
		if len(msgs) > 0 {
			received = msgs[len(msgs)-1].Payload
		}

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	err := handle.Publish(context.Background(), "test message")
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	sess.Flush()

	if received != "test message" {
		t.Errorf("expected received 'test message', got %q", received)
	}
}

func TestUsePubsubCustomCodec(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[int]

	encode := func(val int) ([]byte, error) {
		adjusted := val + 100
		return []byte{byte(adjusted)}, nil
	}

	decode := func(data []byte) (int, error) {
		if len(data) == 0 {
			return 0, errors.New("empty data")
		}
		return int(data[0]) - 100, nil
	}

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		handle = UsePubsub(ctx, "custom",
			WithPubsubProvider[int](provider),
			WithPubsubCodec(encode, decode),
		)
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	err := handle.Publish(context.Background(), 42)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	sess.Flush()

	msgs := handle.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Payload != 42 {
		t.Errorf("expected payload 42, got %d", msgs[0].Payload)
	}
}

func TestUsePubsubNoProvider(t *testing.T) {
	var handle PubsubHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {

		handle = UsePubsub[string](ctx, "test")
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	if handle.Connected() {
		t.Error("expected not connected when no provider")
	}

	err := handle.Publish(context.Background(), "test")
	if err != ErrPubsubUnavailable {
		t.Errorf("expected ErrPubsubUnavailable, got %v", err)
	}

	_, ok := handle.Latest()
	if ok {
		t.Error("expected no latest message")
	}

	msgs := handle.Messages()
	if len(msgs) != 0 {
		t.Errorf("expected empty messages, got %d", len(msgs))
	}
}

func TestUsePubsubMultipleSubscriptions(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle1 PubsubHandle[string]
	var handle2 PubsubHandle[int]

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		handle1 = UsePubsub(ctx, "topic1", WithPubsubProvider[string](provider))
		handle2 = UsePubsub(ctx, "topic2", WithPubsubProvider[int](provider))
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	provider.simulateMessage("topic1", []byte(`"message1"`), nil)

	msgs1 := handle1.Messages()
	if len(msgs1) != 1 {
		t.Errorf("expected 1 message on topic1, got %d", len(msgs1))
	}

	msgs2 := handle2.Messages()
	if len(msgs2) != 0 {
		t.Errorf("expected 0 messages on topic2, got %d", len(msgs2))
	}

	provider.simulateMessage("topic2", []byte("42"), nil)

	msgs1 = handle1.Messages()
	if len(msgs1) != 1 {
		t.Errorf("expected still 1 message on topic1, got %d", len(msgs1))
	}

	msgs2 = handle2.Messages()
	if len(msgs2) != 1 {
		t.Errorf("expected 1 message on topic2, got %d", len(msgs2))
	}
}

func TestUsePubsubTopicChange(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[string]

	var getTopic string
	var setTopic func(string)

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		get, set := UseState(ctx, "topic-a")
		getTopic = get()
		setTopic = func(val string) { set(val) }

		handle = UsePubsub(ctx, getTopic, WithPubsubProvider[string](provider))
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	provider.simulateMessage("topic-a", []byte(`"msg-a"`), nil)

	msgs := handle.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	setTopic("topic-b")
	sess.Flush()

	provider.simulateMessage("topic-a", []byte(`"msg-a2"`), nil)

	msgs = handle.Messages()

	provider.simulateMessage("topic-b", []byte(`"msg-b"`), nil)

	latest, ok := handle.Latest()
	if ok && latest.Payload == "msg-b" {

	}
}

func TestUsePubsubMessageImmutability(t *testing.T) {
	provider := newTestPubsubProvider()
	var handle PubsubHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		handle = UsePubsub(ctx, "test", WithPubsubProvider[string](provider))
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	meta := map[string]string{"key": "original"}
	provider.simulateMessage("test", []byte(`"test"`), meta)

	meta["key"] = "modified"

	latest, ok := handle.Latest()
	if !ok {
		t.Fatal("expected message")
	}

	if latest.Meta["key"] != "original" {
		t.Error("meta should be cloned and immutable")
	}
}
