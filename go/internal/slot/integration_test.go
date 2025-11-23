package slot

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestSlotContext_Integration tests that SlotContext works in a real component session
func TestSlotContext_Integration(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("header", dom.ElementNode("h1").WithChildren(dom.TextNode("Title"))),
			dom.ElementNode("p").WithChildren(dom.TextNode("Body")),
			Slot("footer", dom.ElementNode("button").WithChildren(dom.TextNode("Close"))),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			if !slotCtx.Has(sctx, "header") {
				t.Error("expected header slot to exist")
			}
			if !slotCtx.Has(sctx, "footer") {
				t.Error("expected footer slot to exist")
			}
			if !slotCtx.Has(sctx, "default") {
				t.Error("expected default slot to exist")
			}
			if slotCtx.Has(sctx, "nonexistent") {
				t.Error("expected nonexistent slot to not exist")
			}

			return dom.ElementNode("div").WithChildren(
				slotCtx.Render(sctx, "header"),
				slotCtx.Render(sctx, "default"),
				slotCtx.Render(sctx, "footer"),
			)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSlotContext_EmptyAndMissingSlots tests edge cases
func TestSlotContext_EmptyAndMissingSlots(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("empty"),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			if !slotCtx.Has(sctx, "empty") {
				t.Error("expected empty slot to exist")
			}

			if slotCtx.Has(sctx, "missing") {
				t.Error("expected missing slot to not exist")
			}

			return dom.ElementNode("div").WithChildren(
				slotCtx.Render(sctx, "empty"),
				slotCtx.Render(sctx, "missing"),
			)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSlotContext_DuplicateSlotNames tests that duplicate names append content
func TestSlotContext_DuplicateSlotNames(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("header", dom.ElementNode("h1")),
			Slot("header", dom.ElementNode("h2")),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return dom.ElementNode("div").WithChildren(
				slotCtx.Render(sctx, "header"),
			)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSlotContext_GetSlotNames tests retrieving slot names in declaration order
func TestSlotContext_GetSlotNames(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("header", dom.ElementNode("h1")),
			Slot("footer", dom.ElementNode("button")),
			dom.ElementNode("p").WithChildren(dom.TextNode("Default")),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			names := slotCtx.GetSlotNames(sctx)

			if len(names) != 3 {
				t.Errorf("expected 3 slot names, got %d", len(names))
			}

			expectedOrder := []string{"header", "footer", "default"}
			for i, expected := range expectedOrder {
				if i >= len(names) || names[i] != expected {
					t.Errorf("expected slot at index %d to be %q, got %q", i, expected, names[i])
				}
			}

			return dom.FragmentNode()
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestScopedSlotContext_Integration tests scoped slots with actual data
func TestScopedSlotContext_Integration(t *testing.T) {
	type RowData struct {
		ID   int
		Name string
	}

	slotCtx := CreateScopedSlotContext[RowData]()

	callCount := 0

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			ScopedSlot("row", func(data RowData) *dom.StructuredNode {
				callCount++
				return dom.ElementNode("tr").WithChildren(
					dom.ElementNode("td").WithChildren(dom.TextNode(data.Name)),
				)
			}),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			rows := []RowData{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
			}

			children := make([]*dom.StructuredNode, len(rows))
			for i, row := range rows {
				children[i] = slotCtx.Render(sctx, "row", row)
			}

			return dom.ElementNode("table").WithChildren(children...)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected scoped slot to be called 2 times, got %d", callCount)
	}
}

// TestScopedSlotContext_MissingSlot tests missing scoped slot
func TestScopedSlotContext_MissingSlot(t *testing.T) {
	type Data struct {
		Value string
	}

	slotCtx := CreateScopedSlotContext[Data]()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			result := slotCtx.Render(sctx, "missing", Data{Value: "test"})

			if !result.Fragment {
				t.Error("expected empty fragment for missing scoped slot")
			}

			if slotCtx.Has(sctx, "missing") {
				t.Error("expected Has() to return false for missing slot")
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

// TestScopedSlotContext_Has tests Has method
func TestScopedSlotContext_Has(t *testing.T) {
	type Data struct {
		Value string
	}

	slotCtx := CreateScopedSlotContext[Data]()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			ScopedSlot("provided", func(d Data) *dom.StructuredNode {
				return dom.ElementNode("div")
			}),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			if !slotCtx.Has(sctx, "provided") {
				t.Error("expected Has('provided') to be true")
			}

			if slotCtx.Has(sctx, "missing") {
				t.Error("expected Has('missing') to be false")
			}

			return dom.FragmentNode()
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestScopedSlotContext_NoContextChurn tests that stable scoped slot structure
// doesn't cause unnecessary context changes (uses fingerprinting)
func TestScopedSlotContext_NoContextChurn(t *testing.T) {
	type Data struct {
		Value string
	}

	slotCtx := CreateScopedSlotContext[Data]()

	stableFunc := func(d Data) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode(d.Value))
	}

	renderCount := 0
	innerRenderCount := 0

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		renderCount++

		children := []dom.Item{
			ScopedSlot("item", stableFunc),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			innerRenderCount++
			return slotCtx.Render(sctx, "item", Data{Value: "test"})
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 1 {
		t.Errorf("expected component to render once due to memoization, got %d", renderCount)
	}

	if innerRenderCount != 1 {
		t.Errorf("expected inner render once, got %d (context churn detected)", innerRenderCount)
	}
}

// TestSlotCloningPreventsSharedNodes tests that rendering same slot multiple times
// creates independent node trees (not shared pointers)
func TestSlotCloningPreventsSharedNodes(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("content", dom.ElementNode("div").WithChildren(dom.TextNode("Original"))),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			first := slotCtx.Render(sctx, "content")
			second := slotCtx.Render(sctx, "content")

			if first == second {
				t.Error("expected different node pointers for multiple renders")
			}

			if first.Tag != "div" || second.Tag != "div" {
				t.Error("expected both to be div elements")
			}

			if len(first.Children) > 0 && len(second.Children) > 0 {
				if first.Children[0] == second.Children[0] {
					t.Error("expected children to be independent copies, not shared pointers")
				}
			}

			return dom.ElementNode("container").WithChildren(first, second)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}
