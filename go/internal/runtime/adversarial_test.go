package runtime

import (
	"context"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

// This test demonstrates that child component updates do not propagate into
// the parent tree unless the parent re-renders. The parent copies the child
// node into its children slice, so re-rendering only the child leaves the
// parent tree stale and the session produces no patches.
func TestChildComponentStateUpdateProducesPatch(t *testing.T) {
	var setChildText func(string)

	child := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		value, set := UseState(ctx, "old")
		setChildText = set
		return dom2.ElementNode("span").WithChildren(dom2.TextNode(value()))
	}

	parent := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		childNode := Render(ctx, child, struct{}{})
		return &dom2.StructuredNode{
			Tag:      "div",
			Children: []*dom2.StructuredNode{childNode},
		}
	}

	sess := NewSession(parent, struct{}{})

	var batches [][]dom2diff.Patch
	sess.SetPatchSender(func(patches []dom2diff.Patch) error {
		copyBatch := append([]dom2diff.Patch(nil), patches...)
		batches = append(batches, copyBatch)
		return nil
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	setChildText("new")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after child state change failed: %v", err)
	}

	if len(batches) < 2 {
		t.Fatalf("expected at least two patch batches, got %d", len(batches))
	}
	if len(batches[1]) == 0 {
		t.Fatalf("child update produced no diff; expected setText patch")
	}
}

// This test shows that UsePubsub never calls the provider's Unsubscribe when a
// component using the hook unmounts. The cleanup callback invokes
// unsubscribePubsub, but the provider is never notified, leaving dangling
// subscriptions.
func TestUsePubsubUnsubscribeInvoked(t *testing.T) {
	provider := &mockPubsubProvider{}
	var setVisible func(bool)

	subscriber := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		UsePubsub[int](ctx, "topic", WithPubsubProvider[int](provider))
		return &dom2.StructuredNode{Tag: "span", Text: "sub"}
	}

	root := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		visible, set := UseState(ctx, true)
		setVisible = func(v bool) { set(v) }
		if visible() {
			return Render(ctx, subscriber, struct{}{})
		}
		return &dom2.StructuredNode{Tag: "div", Text: "hidden"}
	}

	sess := NewSession(root, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	setVisible(false)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after unmount failed: %v", err)
	}

	if provider.unsubscribeCalls == 0 {
		t.Fatalf("expected provider.Unsubscribe to be called when component unmounts")
	}
}

type mockPubsubProvider struct {
	unsubscribeCalls int
}

func (m *mockPubsubProvider) Subscribe(ctx context.Context, topic string, handler func([]byte, map[string]string)) (string, error) {
	return "token", nil
}

func (m *mockPubsubProvider) Unsubscribe(ctx context.Context, token string) error {
	m.unsubscribeCalls++
	return nil
}

func (m *mockPubsubProvider) Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
	return nil
}
