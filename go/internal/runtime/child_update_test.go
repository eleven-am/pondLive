package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

func TestChildStateUpdateProducesPatch(t *testing.T) {
	var setChildText func(string)

	child := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		value, set := UseState(ctx, "old")
		setChildText = set
		return dom2.ElementNode("span").WithChildren(dom2.TextNode(value()))
	}

	parent := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div").WithChildren(
			Render(ctx, child, struct{}{}),
		)
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

	second := batches[1]
	if len(second) == 0 {
		t.Fatalf("expected patches for child update, got none")
	}

	foundText := false
	for _, patch := range second {
		if patch.Op == dom2diff.OpSetText {
			foundText = true
		}
	}

	if !foundText {
		t.Fatalf("expected setText patch in second batch, patches: %#v", second)
	}
}
