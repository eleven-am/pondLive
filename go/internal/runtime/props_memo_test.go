package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestChildSkipsRenderWhenPropsSame(t *testing.T) {
	renders := 0

	child := func(ctx Ctx, props string) *dom.StructuredNode {
		renders++
		return dom.ElementNode("span").WithChildren(dom.TextNode(props))
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(
			Render(ctx, child, "hello"),
		)
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("first flush failed: %v", err)
	}
	if renders != 1 {
		t.Fatalf("expected child to render once, got %d", renders)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}
	if renders != 1 {
		t.Fatalf("expected child to skip rerender when props unchanged, got %d", renders)
	}
}

func TestChildRerendersWhenPropsChange(t *testing.T) {
	renders := 0
	var setParentText func(string)

	child := func(ctx Ctx, props string) *dom.StructuredNode {
		renders++
		return dom.ElementNode("span").WithChildren(dom.TextNode(props))
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		text, set := UseState(ctx, "hello")
		setParentText = set
		return dom.ElementNode("div").WithChildren(
			Render(ctx, child, text()),
		)
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("first flush failed: %v", err)
	}
	if renders != 1 {
		t.Fatalf("expected child to render once, got %d", renders)
	}

	setParentText("world")
	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}
	if renders != 2 {
		t.Fatalf("expected child to rerender when props change, got %d", renders)
	}
}
