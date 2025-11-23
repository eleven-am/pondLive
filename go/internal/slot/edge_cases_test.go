package slot

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestNilInputs tests handling of nil inputs
func TestNilInputs(t *testing.T) {
	slotMap := extractSlots(nil)
	if slotMap == nil {
		t.Fatal("expected non-nil SlotMap for nil children")
	}
	if len(slotMap.slots) != 0 {
		t.Errorf("expected 0 slots for nil children, got %d", len(slotMap.slots))
	}

	children := []dom.Item{
		nil,
		Slot("header", dom.ElementNode("h1")),
		nil,
	}
	slotMap = extractSlots(children)
	if slotMap.slots["header"] == nil {
		t.Error("expected header slot despite nil items")
	}
}

// TestEmptyChildren tests various empty children scenarios
func TestEmptyChildren(t *testing.T) {
	testCases := []struct {
		name     string
		children []dom.Item
	}{
		{"nil children", nil},
		{"empty slice", []dom.Item{}},
		{"only nils", []dom.Item{nil, nil, nil}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slotMap := extractSlots(tc.children)
			if slotMap == nil {
				t.Fatal("expected non-nil SlotMap")
			}

			if len(slotMap.slots) != 0 {
				t.Errorf("expected 0 slots, got %d", len(slotMap.slots))
			}
		})
	}
}

// TestSlotContext_WithoutProvide tests using context methods without Provide
func TestSlotContext_WithoutProvide(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		result := slotCtx.Render(ctx, "header")
		if !result.Fragment {
			t.Error("expected empty fragment when used without Provide")
		}

		if slotCtx.Has(ctx, "header") {
			t.Error("expected Has to return false without Provide")
		}

		names := slotCtx.GetSlotNames(ctx)
		if names != nil {
			t.Errorf("expected nil slot names without Provide, got %v", names)
		}

		return dom.FragmentNode()
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestMultipleSlotContexts tests using multiple independent slot contexts
func TestMultipleSlotContexts(t *testing.T) {
	slotCtx1 := CreateSlotContext()
	slotCtx2 := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children1 := []dom.Item{
			Slot("header", dom.ElementNode("h1").WithChildren(dom.TextNode("Context 1"))),
		}

		children2 := []dom.Item{
			Slot("header", dom.ElementNode("h2").WithChildren(dom.TextNode("Context 2"))),
		}

		return dom.ElementNode("div").WithChildren(
			slotCtx1.Provide(ctx, children1, func(sctx1 runtime.Ctx) *dom.StructuredNode {
				return slotCtx2.Provide(sctx1, children2, func(sctx2 runtime.Ctx) *dom.StructuredNode {

					header1 := slotCtx1.Render(sctx2, "header")
					header2 := slotCtx2.Render(sctx2, "header")

					if header1.Tag != "h1" {
						t.Errorf("expected h1 from context1, got %s", header1.Tag)
					}
					if header2.Tag != "h2" {
						t.Errorf("expected h2 from context2, got %s", header2.Tag)
					}

					return dom.ElementNode("section").WithChildren(header1, header2)
				})
			}),
		)
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestNestedSlotContexts tests nested usage of same context
func TestNestedSlotContexts(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		outerChildren := []dom.Item{
			Slot("outer", dom.ElementNode("div").WithChildren(dom.TextNode("Outer"))),
		}

		return slotCtx.Provide(ctx, outerChildren, func(outerCtx runtime.Ctx) *dom.StructuredNode {

			innerChildren := []dom.Item{
				Slot("inner", dom.ElementNode("span").WithChildren(dom.TextNode("Inner"))),
			}

			return slotCtx.Provide(outerCtx, innerChildren, func(innerCtx runtime.Ctx) *dom.StructuredNode {

				if !slotCtx.Has(innerCtx, "inner") {
					t.Error("expected inner slot to exist")
				}

				if slotCtx.Has(innerCtx, "outer") {
					t.Error("expected outer slot to NOT exist in inner context")
				}

				return slotCtx.Render(innerCtx, "inner")
			})
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSetFallback_AfterSlotProvided tests that fallbacks are ignored when slot is provided
func TestSetFallback_AfterSlotProvided(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("header", dom.ElementNode("h1").WithChildren(dom.TextNode("Provided"))),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			slotCtx.SetFallback(sctx, "header",
				dom.ElementNode("h2").WithChildren(dom.TextNode("Fallback")),
			)

			result := slotCtx.Render(sctx, "header")
			if result.Tag != "h1" {
				t.Errorf("expected provided h1, got %s", result.Tag)
			}

			return result
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSetFallback_ForMissingSlot tests fallback when slot not provided
func TestSetFallback_ForMissingSlot(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			slotCtx.SetFallback(sctx, "footer",
				dom.ElementNode("button").WithChildren(dom.TextNode("Default Close")),
			)

			result := slotCtx.Render(sctx, "footer")
			if result.Tag != "button" {
				t.Errorf("expected fallback button, got %s", result.Tag)
			}

			return result
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestComplexNestedSlots tests deeply nested slot content
func TestComplexNestedSlots(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		deepNested := dom.ElementNode("div").WithChildren(
			dom.ElementNode("div").WithChildren(
				dom.ElementNode("div").WithChildren(
					dom.ElementNode("span").WithChildren(dom.TextNode("Deep")),
				),
			),
		)

		children := []dom.Item{
			Slot("nested", deepNested),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "nested")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestMixedContentInSlot tests slot with mixed node types
func TestMixedContentInSlot(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("mixed",
				dom.TextNode("Text "),
				dom.ElementNode("strong").WithChildren(dom.TextNode("Bold")),
				dom.TextNode(" More text"),
				dom.ElementNode("em").WithChildren(dom.TextNode("Italic")),
			),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "mixed")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestScopedSlot_WithNilData tests scoped slot with nil-like data
func TestScopedSlot_WithNilData(t *testing.T) {
	type Data struct {
		Value *string
	}

	slotCtx := CreateScopedSlotContext[Data]()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			ScopedSlot("item", func(d Data) *dom.StructuredNode {
				text := "nil"
				if d.Value != nil {
					text = *d.Value
				}
				return dom.ElementNode("div").WithChildren(dom.TextNode(text))
			}),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			result := slotCtx.Render(sctx, "item", Data{Value: nil})
			if len(result.Children) == 0 {
				t.Fatal("expected rendered content")
			}
			if result.Children[0].Text != "nil" {
				t.Errorf("expected 'nil' text, got %q", result.Children[0].Text)
			}

			return result
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestScopedSlot_ZeroValue tests scoped slot with zero values
func TestScopedSlot_ZeroValue(t *testing.T) {
	type Data struct {
		Count int
		Name  string
		Flag  bool
	}

	slotCtx := CreateScopedSlotContext[Data]()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		called := false
		children := []dom.Item{
			ScopedSlot("item", func(d Data) *dom.StructuredNode {
				called = true

				if d.Count != 0 || d.Name != "" || d.Flag != false {
					t.Errorf("expected zero values, got Count=%d, Name=%q, Flag=%v", d.Count, d.Name, d.Flag)
				}
				return dom.ElementNode("div")
			}),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			result := slotCtx.Render(sctx, "item", Data{})
			if !called {
				t.Error("expected scoped slot function to be called")
			}
			return result
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestMetadataNotLeaked tests that slot metadata doesn't leak to rendered output
func TestMetadataNotLeaked(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("header", dom.ElementNode("h1")),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			result := slotCtx.Render(sctx, "header")

			if result.Metadata != nil {
				if _, hasSlotMeta := result.Metadata[slotNameKey]; hasSlotMeta {
					t.Error("slot metadata leaked to rendered output")
				}
			}

			return result
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSlotWithFragmentChildren tests slot containing fragments
func TestSlotWithFragmentChildren(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		fragment := dom.FragmentNode()
		fragment.Children = []*dom.StructuredNode{
			dom.ElementNode("span").WithChildren(dom.TextNode("One")),
			dom.ElementNode("span").WithChildren(dom.TextNode("Two")),
		}

		children := []dom.Item{
			Slot("header", fragment),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "header")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}
