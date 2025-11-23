package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// This test mirrors the tab scenario in a minimal form:
// switching between two child components should produce patches.
func TestComponentWrapperSwitchProducesPatches(t *testing.T) {
	childA := Component[struct{}](func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode("A"))
	})
	childB := Component[struct{}](func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode("B"))
	})

	var setActive func(string)

	root := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		active, set := UseState(ctx, "a")
		setActive = set

		var chosen Component[struct{}]
		if active() == "a" {
			chosen = childA
		} else {
			chosen = childB
		}

		return Render(ctx, chosen, struct{}{}, WithKey("content"))
	}

	sess := NewSession(root, struct{}{})
	sess.SetPatchSender(func(p []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	prev := sess.Tree().Flatten()

	setActive("b")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after switch failed: %v", err)
	}
	next := sess.Tree().Flatten()

	patches := dom2diff.Diff(prev, next)
	if len(patches) == 0 {
		t.Fatalf("expected patches when switching child component; prev=%q next=%q", prev.ToHTML(), next.ToHTML())
	}
}
