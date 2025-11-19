package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

func TestPropsOptimizationComprehensive(t *testing.T) {
	parentRenders := 0
	child1Renders := 0
	child2Renders := 0

	child1 := func(ctx Ctx, props struct{ Value string }) *dom2.StructuredNode {
		child1Renders++
		return dom2.ElementNode("span").WithChildren(dom2.TextNode(props.Value))
	}

	child2 := func(ctx Ctx, props struct{ Value string }) *dom2.StructuredNode {
		child2Renders++
		return dom2.ElementNode("span").WithChildren(dom2.TextNode(props.Value))
	}

	var setParentCount func(int)
	parent := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		parentRenders++
		count, set := UseState(ctx, 0)
		setParentCount = set

		return dom2.ElementNode("div").WithChildren(
			Render(ctx, child1, struct{ Value string }{Value: "static"}),
			Render(ctx, child2, struct{ Value string }{Value: string(rune('0' + count()))}),
		)
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	t.Log("=== Initial render ===")
	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	t.Logf("Parent renders: %d, Child1 renders: %d, Child2 renders: %d", parentRenders, child1Renders, child2Renders)

	if parentRenders != 1 || child1Renders != 1 || child2Renders != 1 {
		t.Fatalf("expected all components to render once initially, got parent=%d, child1=%d, child2=%d",
			parentRenders, child1Renders, child2Renders)
	}

	t.Log("=== Parent re-renders with same child1 props, different child2 props ===")
	setParentCount(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}
	t.Logf("Parent renders: %d, Child1 renders: %d, Child2 renders: %d", parentRenders, child1Renders, child2Renders)

	if parentRenders != 2 {
		t.Errorf("expected parent to render twice, got %d", parentRenders)
	}
	if child1Renders != 1 {
		t.Errorf("expected child1 to skip render (props unchanged), got %d renders", child1Renders)
	}
	if child2Renders != 2 {
		t.Errorf("expected child2 to re-render (props changed), got %d renders", child2Renders)
	}

	t.Log("=== Parent re-renders again with different value ===")
	setParentCount(2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("third flush failed: %v", err)
	}
	t.Logf("Parent renders: %d, Child1 renders: %d, Child2 renders: %d", parentRenders, child1Renders, child2Renders)

	if parentRenders != 3 {
		t.Errorf("expected parent to render three times, got %d", parentRenders)
	}
	if child1Renders != 1 {
		t.Errorf("expected child1 to still have 1 render (props still unchanged), got %d", child1Renders)
	}
	if child2Renders != 3 {
		t.Errorf("expected child2 to re-render again (props changed), got %d renders", child2Renders)
	}
}
