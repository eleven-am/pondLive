package runtime

import (
	"context"
	"errors"
	"testing"
)

type mockProvider struct {
	publishFn func(string, []byte, map[string]string) error
}

func (m *mockProvider) Subscribe(*LiveSession, string, PubsubHandler) (string, error) {
	return "", nil
}

func (m *mockProvider) Unsubscribe(*LiveSession, string) error {
	return nil
}

func (m *mockProvider) Publish(topic string, payload []byte, meta map[string]string) error {
	if m.publishFn != nil {
		return m.publishFn(topic, payload, meta)
	}
	return nil
}

func TestPubsubPublisherNilProvider(t *testing.T) {
	adapter := WrapPubsubProvider(nil)
	if adapter == nil {
		t.Fatal("expected adapter")
	}
	if err := adapter(context.Background(), "topic", nil, nil); err != ErrPubsubUnavailable {
		t.Fatalf("expected ErrPubsubUnavailable, got %v", err)
	}
}

func TestPubsubPublisherContextCancel(t *testing.T) {
	provider := &mockProvider{}
	adapter := WrapPubsubProvider(provider)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := adapter(ctx, "topic", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestPubsubPublisherDelegates(t *testing.T) {
	var (
		gotTopic   string
		gotPayload []byte
		gotMeta    map[string]string
	)
	provider := &mockProvider{
		publishFn: func(topic string, payload []byte, meta map[string]string) error {
			gotTopic = topic
			gotPayload = append([]byte(nil), payload...)
			gotMeta = meta
			return nil
		},
	}
	adapter := WrapPubsubProvider(provider)
	payload := []byte("data")
	meta := map[string]string{"k": "v"}
	if err := adapter(context.Background(), "news", payload, meta); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if gotTopic != "news" {
		t.Fatalf("unexpected topic %q", gotTopic)
	}
	if string(gotPayload) != "data" {
		t.Fatalf("unexpected payload %q", string(gotPayload))
	}
	if gotMeta["k"] != "v" {
		t.Fatalf("unexpected meta %v", gotMeta)
	}
	payload[0] = 'X'
	meta["k"] = "z"
	if string(gotPayload) != "data" {
		t.Fatalf("payload mutated after publish")
	}
	if gotMeta["k"] != "v" {
		t.Fatalf("meta mutated after publish: %q", gotMeta["k"])
	}
}
