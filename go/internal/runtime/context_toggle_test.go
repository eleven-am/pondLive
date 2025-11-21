package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestContextProviderToggle(t *testing.T) {
	ctx := CreateContext("A")
	var seen []string
	var setValue func(string)

	child := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		seen = append(seen, ctx.Use(rctx))
		return dom.ElementNode("span")
	}

	parent := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		value, set := UseState(rctx, "A")
		setValue = set
		return ctx.Provide(rctx, value(), func(pctx Ctx) *dom.StructuredNode {
			return Render(pctx, child, struct{}{})
		})
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush 1 failed: %v", err)
	}

	setValue("B")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush 2 failed: %v", err)
	}

	if len(seen) != 2 || seen[0] != "A" || seen[1] != "B" {
		t.Fatalf("expected context values [A B], got %v", seen)
	}
}

func TestMultipleContextFlip(t *testing.T) {
	colorCtx := CreateContext("red")
	sizeCtx := CreateContext(10)
	var snapshots []struct {
		color string
		size  int
	}
	var setColor func(string)

	leaf := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		snapshots = append(snapshots, struct {
			color string
			size  int
		}{
			color: colorCtx.Use(rctx),
			size:  sizeCtx.Use(rctx),
		})
		return dom.ElementNode("span")
	}

	parent := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		color, set := UseState(rctx, "red")
		setColor = set
		return colorCtx.Provide(rctx, color(), func(pctx Ctx) *dom.StructuredNode {
			return sizeCtx.Provide(pctx, 20, func(qctx Ctx) *dom.StructuredNode {
				return Render(qctx, leaf, struct{}{})
			})
		})
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush 1 failed: %v", err)
	}

	setColor("blue")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush 2 failed: %v", err)
	}

	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].color != "red" || snapshots[0].size != 20 {
		t.Fatalf("snapshot[0] unexpected: %+v", snapshots[0])
	}
	if snapshots[1].color != "blue" || snapshots[1].size != 20 {
		t.Fatalf("snapshot[1] unexpected: %+v", snapshots[1])
	}
}
