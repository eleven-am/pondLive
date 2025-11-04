package runtime

import (
	"context"
	"sync"
	"testing"
)

func TestNewPubsubPublishUsesPublisher(t *testing.T) {
	var (
		mu      sync.Mutex
		topic   string
		payload []byte
		meta    map[string]string
	)

	publisher := PubsubPublishFunc(func(ctx context.Context, t string, data []byte, m map[string]string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		mu.Lock()
		topic = t
		payload = append([]byte(nil), data...)
		meta = m
		mu.Unlock()
		return nil
	})

	ps := NewPubsub[string]("news", publisher)

	inputMeta := map[string]string{"id": "123"}
	if err := ps.Publish(context.Background(), "hello", inputMeta); err != nil {
		t.Fatalf("publish: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if topic != "news" {
		t.Fatalf("expected topic news, got %q", topic)
	}
	if string(payload) != "\"hello\"" {
		t.Fatalf("expected JSON payload, got %q", string(payload))
	}
	if meta == nil || meta["id"] != "123" {
		t.Fatalf("expected metadata copy, got %#v", meta)
	}
	inputMeta["id"] = "456"
	if meta["id"] != "123" {
		t.Fatalf("metadata mutated after publish: %q", meta["id"])
	}
}

func TestNewPubsubWithoutPublisher(t *testing.T) {
	ps := NewPubsub[string]("news", nil)
	if err := ps.Publish(context.Background(), "hello", nil); err != ErrPubsubUnavailable {
		t.Fatalf("expected ErrPubsubUnavailable, got %v", err)
	}
}

func TestNewPubsubPublishHonorsContextCancellation(t *testing.T) {
	publisher := PubsubPublishFunc(func(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
		<-ctx.Done()
		return ctx.Err()
	})

	ps := NewPubsub[string]("news", publisher)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := ps.Publish(ctx, "hello", nil); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestNewPubsubFallsBackToDefaultPublisher(t *testing.T) {
	original := loadDefaultPubsubPublisher()
	defer SetDefaultPubsubPublisher(original)

	var (
		mu       sync.Mutex
		recorded string
	)

	SetDefaultPubsubPublisher(PubsubPublishFunc(func(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
		mu.Lock()
		recorded = topic + ":" + string(payload)
		mu.Unlock()
		return nil
	}))

	ps := NewPubsub[string]("updates", nil)
	if err := ps.Publish(context.Background(), "hello", nil); err != nil {
		t.Fatalf("publish with default: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if recorded != "updates:\"hello\"" {
		t.Fatalf("unexpected recorded value %q", recorded)
	}
}
