package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// TestMultipleComponentInstancesWithoutKeys tests that rendering the same component
// multiple times without explicit keys works correctly using positional index-based keys.
func TestMultipleComponentInstancesWithoutKeys(t *testing.T) {
	var inputRenderCount int

	input := func(ctx Ctx, props struct{ Name string }) *dom.StructuredNode {
		inputRenderCount++
		return dom.ElementNode("input").WithAttr("name", props.Name)
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {

		input1 := Render(ctx, input, struct{ Name string }{Name: "email"})
		input2 := Render(ctx, input, struct{ Name string }{Name: "password"})

		return &dom.StructuredNode{
			Tag: "div",
			Children: []*dom.StructuredNode{
				dom.ElementNode("h1").WithChildren(dom.TextNode("Test")),
				input1,
				input2,
			},
		}
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	inputRenderCount = 0
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if inputRenderCount != 2 {
		t.Errorf("Expected input to render 2 times, got %d", inputRenderCount)
	}

	t.Logf("Successfully rendered %d input components without explicit keys", inputRenderCount)
}

// TestMultipleComponentInstancesStateIndependence ensures that when the same component
// is rendered multiple times without keys, each instance maintains independent state.
func TestMultipleComponentInstancesStateIndependence(t *testing.T) {
	var getCount1 func() int
	var setCount1 func(int)
	var getCount2 func() int
	var setCount2 func(int)

	var counter1RenderCount int
	var counter2RenderCount int

	counter := func(ctx Ctx, props struct{ Label string }) *dom.StructuredNode {
		count, setCount := UseState(ctx, 0)

		if props.Label == "Counter 1" {
			getCount1 = count
			setCount1 = setCount
			counter1RenderCount++
		} else if props.Label == "Counter 2" {
			getCount2 = count
			setCount2 = setCount
			counter2RenderCount++
		}

		return dom.ElementNode("div").WithChildren(
			dom.ElementNode("span").WithChildren(dom.TextNode(props.Label)),
			dom.ElementNode("span").WithAttr("data-count", string(rune('0'+count()))),
		)
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {

		counter1 := Render(ctx, counter, struct{ Label string }{Label: "Counter 1"})
		counter2 := Render(ctx, counter, struct{ Label string }{Label: "Counter 2"})

		return &dom.StructuredNode{
			Tag:      "div",
			Children: []*dom.StructuredNode{counter1, counter2},
		}
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	if getCount1() != 0 {
		t.Errorf("Expected counter1 to be 0, got %d", getCount1())
	}
	if getCount2() != 0 {
		t.Errorf("Expected counter2 to be 0, got %d", getCount2())
	}

	setCount1(5)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after counter1 update failed: %v", err)
	}

	if getCount1() != 5 {
		t.Errorf("Expected counter1 to be 5, got %d", getCount1())
	}
	if getCount2() != 0 {
		t.Errorf("Expected counter2 to still be 0, got %d", getCount2())
	}

	setCount2(10)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after counter2 update failed: %v", err)
	}

	if getCount1() != 5 {
		t.Errorf("Expected counter1 to still be 5, got %d", getCount1())
	}
	if getCount2() != 10 {
		t.Errorf("Expected counter2 to be 10, got %d", getCount2())
	}

	t.Logf("Counter 1 rendered %d times, Counter 2 rendered %d times", counter1RenderCount, counter2RenderCount)
	t.Logf("Successfully maintained independent state for multiple instances")
}
