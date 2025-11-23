package slot

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestManySlots tests handling of many named slots
func TestManySlots(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := make([]dom.Item, 100)
		for i := 0; i < 100; i++ {
			slotName := fmt.Sprintf("slot_%d", i)
			children[i] = Slot(slotName,
				dom.ElementNode("div").WithChildren(dom.TextNode(slotName)),
			)
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			for i := 0; i < 100; i++ {
				slotName := fmt.Sprintf("slot_%d", i)
				if !slotCtx.Has(sctx, slotName) {
					t.Errorf("expected slot %s to exist", slotName)
				}
			}

			rendered := make([]*dom.StructuredNode, 100)
			for i := 0; i < 100; i++ {
				slotName := fmt.Sprintf("slot_%d", i)
				rendered[i] = slotCtx.Render(sctx, slotName)
			}

			return dom.ElementNode("div").WithChildren(rendered...)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestManyChildrenInSlot tests slot with many children
func TestManyChildrenInSlot(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		slotChildren := make([]dom.Item, 1000)
		for i := 0; i < 1000; i++ {
			slotChildren[i] = dom.ElementNode("span").WithChildren(
				dom.TextNode(fmt.Sprintf("Child %d", i)),
			)
		}

		children := []dom.Item{
			Slot("many", slotChildren...),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "many")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestDeeplyNestedSlotContexts tests deep nesting of slot contexts
func TestDeeplyNestedSlotContexts(t *testing.T) {
	contexts := make([]*SlotContext, 20)
	for i := range contexts {
		contexts[i] = CreateSlotContext()
	}

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		var buildNested func(depth int, sctx runtime.Ctx) *dom.StructuredNode
		buildNested = func(depth int, sctx runtime.Ctx) *dom.StructuredNode {
			if depth >= 20 {
				return dom.ElementNode("div").WithChildren(dom.TextNode("Bottom"))
			}

			children := []dom.Item{
				Slot("slot", dom.ElementNode("span").WithChildren(
					dom.TextNode(fmt.Sprintf("Level %d", depth)),
				)),
			}

			return contexts[depth].Provide(sctx, children, func(innerCtx runtime.Ctx) *dom.StructuredNode {
				return dom.ElementNode("div").WithChildren(
					contexts[depth].Render(innerCtx, "slot"),
					buildNested(depth+1, innerCtx),
				)
			})
		}

		return buildNested(0, ctx)
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestScopedSlot_ManyInvocations tests scoped slot called many times
func TestScopedSlot_ManyInvocations(t *testing.T) {
	type Item struct {
		ID int
	}

	slotCtx := CreateScopedSlotContext[Item]()

	callCount := 0

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			ScopedSlot("item", func(item Item) *dom.StructuredNode {
				callCount++
				return dom.ElementNode("div").WithChildren(
					dom.TextNode(fmt.Sprintf("Item %d", item.ID)),
				)
			}),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {

			items := make([]*dom.StructuredNode, 1000)
			for i := 0; i < 1000; i++ {
				items[i] = slotCtx.Render(sctx, "item", Item{ID: i})
			}

			return dom.ElementNode("div").WithChildren(items...)
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if callCount != 1000 {
		t.Errorf("expected 1000 invocations, got %d", callCount)
	}
}

// TestLargeSlotContent tests slot with very large content
func TestLargeSlotContent(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		largeText := strings.Repeat("Lorem ipsum dolor sit amet ", 400)

		children := []dom.Item{
			Slot("large", dom.ElementNode("div").WithChildren(dom.TextNode(largeText))),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "large")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestSlotWithDeeplyNestedDOM tests slot containing deeply nested DOM
func TestSlotWithDeeplyNestedDOM(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		var buildNested func(depth int) *dom.StructuredNode
		buildNested = func(depth int) *dom.StructuredNode {
			if depth >= 50 {
				return dom.TextNode("Deep")
			}
			return dom.ElementNode("div").WithChildren(buildNested(depth + 1))
		}

		children := []dom.Item{
			Slot("deep", buildNested(0)),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "deep")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestDuplicateSlotAppending tests massive duplication of same slot name
func TestDuplicateSlotAppending(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		children := make([]dom.Item, 100)
		for i := 0; i < 100; i++ {
			children[i] = Slot("header",
				dom.ElementNode("span").WithChildren(dom.TextNode(fmt.Sprintf("Header %d", i))),
			)
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			result := slotCtx.Render(sctx, "header")

			if !result.Fragment {
				t.Fatal("expected fragment for multiple nodes")
			}
			if len(result.Children) != 100 {
				t.Errorf("expected 100 children, got %d", len(result.Children))
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

// TestManyGetSlotNamesCalls tests repeated GetSlotNames calls
func TestManyGetSlotNamesCalls(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("a", dom.ElementNode("div")),
			Slot("b", dom.ElementNode("div")),
			Slot("c", dom.ElementNode("div")),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			for i := 0; i < 1000; i++ {
				names := slotCtx.GetSlotNames(sctx)
				if len(names) != 3 {
					t.Errorf("iteration %d: expected 3 names, got %d", i, len(names))
					break
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

// TestComplexMixedScenario tests complex real-world-like scenario
func TestComplexMixedScenario(t *testing.T) {

	regularSlotCtx := CreateSlotContext()

	type RowData struct {
		ID    int
		Value string
	}
	scopedSlotCtx := CreateScopedSlotContext[RowData]()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {

		regularChildren := []dom.Item{
			Slot("header", dom.ElementNode("h1").WithChildren(dom.TextNode("Title"))),
			Slot("sidebar", dom.ElementNode("nav").WithChildren(dom.TextNode("Nav"))),
			dom.ElementNode("p").WithChildren(dom.TextNode("Default content")),
		}

		scopedChildren := []dom.Item{
			ScopedSlot("row", func(row RowData) *dom.StructuredNode {
				return dom.ElementNode("tr").WithChildren(
					dom.ElementNode("td").WithChildren(dom.TextNode(fmt.Sprintf("%d", row.ID))),
					dom.ElementNode("td").WithChildren(dom.TextNode(row.Value)),
				)
			}),
		}

		return regularSlotCtx.Provide(ctx, regularChildren, func(rctx runtime.Ctx) *dom.StructuredNode {
			return scopedSlotCtx.Provide(rctx, scopedChildren, func(sctx runtime.Ctx) *dom.StructuredNode {

				rows := make([]*dom.StructuredNode, 50)
				for i := 0; i < 50; i++ {
					rows[i] = scopedSlotCtx.Render(sctx, "row", RowData{
						ID:    i,
						Value: fmt.Sprintf("Value %d", i),
					})
				}

				table := dom.ElementNode("table").WithChildren(rows...)

				return dom.ElementNode("div").WithChildren(
					regularSlotCtx.Render(sctx, "header"),
					regularSlotCtx.Render(sctx, "sidebar"),
					table,
					regularSlotCtx.Render(sctx, "default"),
				)
			})
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

// TestRepeatedFlushes tests multiple flush cycles don't break slots
func TestRepeatedFlushes(t *testing.T) {
	slotCtx := CreateSlotContext()

	renderFn := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		children := []dom.Item{
			Slot("header", dom.ElementNode("h1").WithChildren(dom.TextNode("Header"))),
		}

		return slotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
			return slotCtx.Render(sctx, "header")
		})
	}

	sess := runtime.NewSession(renderFn, struct{}{})
	sess.SetPatchSender(func(patches []diff.Patch) error { return nil })

	for i := 0; i < 100; i++ {
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}
}

// BenchmarkSlotExtraction benchmarks slot extraction performance
func BenchmarkSlotExtraction(b *testing.B) {
	children := []dom.Item{
		Slot("header", dom.ElementNode("h1")),
		Slot("body", dom.ElementNode("div")),
		Slot("footer", dom.ElementNode("footer")),
		dom.ElementNode("p").WithChildren(dom.TextNode("Default")),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractSlots(children)
	}
}

// BenchmarkSlotRendering benchmarks slot rendering
func BenchmarkSlotRendering(b *testing.B) {
	content := &SlotContent{
		nodes: []*dom.StructuredNode{
			dom.ElementNode("h1"),
			dom.ElementNode("p"),
			dom.ElementNode("div"),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderSlotContent(content)
	}
}

// BenchmarkScopedSlotInvocation benchmarks scoped slot function calls
func BenchmarkScopedSlotInvocation(b *testing.B) {
	type Data struct {
		Value string
	}

	fn := func(d Data) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode(d.Value))
	}

	data := Data{Value: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fn(data)
	}
}
