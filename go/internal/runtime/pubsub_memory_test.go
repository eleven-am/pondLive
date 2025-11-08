package runtime

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
)

func TestInMemoryPubsubPublish(t *testing.T) {
	provider := NewInMemoryPubsubProvider()
	sess := NewLiveSession(SessionID("sess"), 1, func(Ctx, struct{}) h.Node { return h.Fragment() }, struct{}{}, nil)

	payloads := make(chan string, 2)

	token, err := provider.Subscribe(sess, "updates", func(_ string, payload []byte, _ map[string]string) {
		payloads <- string(payload)
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token")
	}

	if err := provider.Publish("updates", []byte("hello"), nil); err != nil {
		t.Fatalf("publish: %v", err)
	}

	got := <-payloads
	if got != "hello" {
		t.Fatalf("unexpected payload %q", got)
	}

	if err := provider.Unsubscribe(sess, token); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}

	_ = provider.Publish("updates", []byte("ignored"), nil)
	select {
	case <-payloads:
		t.Fatalf("expected no further payloads after unsubscribe")
	default:
	}
}
