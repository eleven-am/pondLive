package slot

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// TestSlotMarkerCreation tests that Slot creates valid markers
func TestSlotMarkerCreation(t *testing.T) {
	marker := Slot("header",
		dom.ElementNode("h1").WithChildren(dom.TextNode("Title")),
		dom.ElementNode("p").WithChildren(dom.TextNode("Subtitle")),
	)

	parent := dom.FragmentNode()
	marker.ApplyTo(parent)

	if len(parent.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(parent.Children))
	}

	fragment := parent.Children[0]

	if fragment.Metadata == nil {
		t.Fatal("expected metadata to be set")
	}

	slotName, ok := fragment.Metadata[slotNameKey].(string)
	if !ok {
		t.Fatal("expected slot:name metadata")
	}

	if slotName != "header" {
		t.Errorf("expected slot name 'header', got %q", slotName)
	}

	if len(fragment.Children) != 2 {
		t.Errorf("expected 2 children in fragment, got %d", len(fragment.Children))
	}
}

// TestExtractSlots tests slot extraction from children
func TestExtractSlots(t *testing.T) {
	children := []dom.Item{

		Slot("header",
			dom.ElementNode("h1").WithChildren(dom.TextNode("Header")),
		),

		dom.ElementNode("p").WithChildren(dom.TextNode("Body")),

		Slot("footer",
			dom.ElementNode("button").WithChildren(dom.TextNode("Close")),
		),

		dom.ElementNode("div").WithChildren(dom.TextNode("More body")),
	}

	slotMap := extractSlots(children)

	if slotMap.slots["header"] == nil {
		t.Fatal("expected header slot to exist")
	}
	if len(slotMap.slots["header"].nodes) != 1 {
		t.Errorf("expected 1 node in header slot, got %d", len(slotMap.slots["header"].nodes))
	}
	if slotMap.slots["header"].nodes[0].Tag != "h1" {
		t.Errorf("expected h1 in header slot, got %s", slotMap.slots["header"].nodes[0].Tag)
	}

	if slotMap.slots["footer"] == nil {
		t.Fatal("expected footer slot to exist")
	}
	if len(slotMap.slots["footer"].nodes) != 1 {
		t.Errorf("expected 1 node in footer slot, got %d", len(slotMap.slots["footer"].nodes))
	}

	if slotMap.slots["default"] == nil {
		t.Fatal("expected default slot to exist")
	}
	if len(slotMap.slots["default"].nodes) != 2 {
		t.Errorf("expected 2 nodes in default slot, got %d", len(slotMap.slots["default"].nodes))
	}
}

// TestExtractSlots_DuplicateSlotNames tests that duplicate slot names append
func TestExtractSlots_DuplicateSlotNames(t *testing.T) {
	children := []dom.Item{
		Slot("header", dom.ElementNode("h1").WithChildren(dom.TextNode("Title"))),
		Slot("header", dom.ElementNode("p").WithChildren(dom.TextNode("Subtitle"))),
	}

	slotMap := extractSlots(children)

	if slotMap.slots["header"] == nil {
		t.Fatal("expected header slot to exist")
	}

	if len(slotMap.slots["header"].nodes) != 2 {
		t.Fatalf("expected 2 nodes in header slot, got %d", len(slotMap.slots["header"].nodes))
	}

	if slotMap.slots["header"].nodes[0].Tag != "h1" {
		t.Errorf("expected first node to be h1, got %s", slotMap.slots["header"].nodes[0].Tag)
	}

	if slotMap.slots["header"].nodes[1].Tag != "p" {
		t.Errorf("expected second node to be p, got %s", slotMap.slots["header"].nodes[1].Tag)
	}
}

// TestExtractSlots_EmptySlot tests empty slot handling
func TestExtractSlots_EmptySlot(t *testing.T) {
	children := []dom.Item{
		Slot("header"),
		dom.ElementNode("p").WithChildren(dom.TextNode("Body")),
	}

	slotMap := extractSlots(children)

	if slotMap.slots["header"] == nil {
		t.Fatal("expected header slot to exist even if empty")
	}

	if len(slotMap.slots["header"].nodes) != 0 {
		t.Errorf("expected empty header slot, got %d nodes", len(slotMap.slots["header"].nodes))
	}
}

// TestRenderSlotContent tests slot content rendering
func TestRenderSlotContent(t *testing.T) {

	result := renderSlotContent(nil)
	if !result.Fragment {
		t.Error("expected empty fragment for nil content")
	}

	result = renderSlotContent(&SlotContent{nodes: nil})
	if !result.Fragment {
		t.Error("expected empty fragment for empty content")
	}

	content := &SlotContent{
		nodes: []*dom.StructuredNode{
			dom.ElementNode("div").WithChildren(dom.TextNode("test")),
		},
	}
	result = renderSlotContent(content)
	if result.Tag != "div" {
		t.Errorf("expected single node to be returned directly, got tag %q", result.Tag)
	}

	content = &SlotContent{
		nodes: []*dom.StructuredNode{
			dom.ElementNode("h1"),
			dom.ElementNode("p"),
		},
	}
	result = renderSlotContent(content)
	if !result.Fragment {
		t.Error("expected fragment for multiple nodes")
	}
	if len(result.Children) != 2 {
		t.Errorf("expected 2 children in fragment, got %d", len(result.Children))
	}
}

// TestScopedSlotExtraction tests scoped slot function extraction
func TestScopedSlotExtraction(t *testing.T) {
	type TestData struct {
		Value string
	}

	called := false
	fn := func(data TestData) *dom.StructuredNode {
		called = true
		return dom.ElementNode("div").WithChildren(dom.TextNode(data.Value))
	}

	children := []dom.Item{
		ScopedSlot("item", fn),
	}

	slotMap := extractScopedSlots[TestData](children)

	fns := slotMap.slots["item"]
	if fns == nil || len(fns) == 0 {
		t.Fatal("expected item scoped slot to exist")
	}

	if len(fns) != 1 {
		t.Fatalf("expected 1 function, got %d", len(fns))
	}

	result := fns[0](TestData{Value: "test"})

	if !called {
		t.Error("expected function to be called")
	}

	if result.Tag != "div" {
		t.Errorf("expected div, got %s", result.Tag)
	}

	if len(result.Children) == 0 {
		t.Fatal("expected children in result")
	}

	if result.Children[0].Text != "test" {
		t.Errorf("expected text 'test', got %q", result.Children[0].Text)
	}
}

// TestScopedSlotExtraction_Duplicates tests that duplicate scoped slot names append
func TestScopedSlotExtraction_Duplicates(t *testing.T) {
	type TestData struct {
		Value string
	}

	callOrder := []string{}

	fn1 := func(data TestData) *dom.StructuredNode {
		callOrder = append(callOrder, "fn1")
		return dom.ElementNode("div").WithChildren(dom.TextNode("First"))
	}

	fn2 := func(data TestData) *dom.StructuredNode {
		callOrder = append(callOrder, "fn2")
		return dom.ElementNode("span").WithChildren(dom.TextNode("Second"))
	}

	children := []dom.Item{
		ScopedSlot("item", fn1),
		ScopedSlot("item", fn2),
	}

	slotMap := extractScopedSlots[TestData](children)

	fns := slotMap.slots["item"]
	if len(fns) != 2 {
		t.Fatalf("expected 2 functions for duplicate names, got %d", len(fns))
	}

	result1 := fns[0](TestData{Value: "test"})
	result2 := fns[1](TestData{Value: "test"})

	if len(callOrder) != 2 {
		t.Errorf("expected both functions to be called, got %d calls", len(callOrder))
	}

	if result1.Tag != "div" || result1.Children[0].Text != "First" {
		t.Error("first function result incorrect")
	}
	if result2.Tag != "span" || result2.Children[0].Text != "Second" {
		t.Error("second function result incorrect")
	}
}

// TestNodeToSlotContent tests conversion from node to slot content
func TestNodeToSlotContent(t *testing.T) {

	content := nodeToSlotContent(nil)
	if content == nil || len(content.nodes) != 0 {
		t.Error("expected empty SlotContent for nil node")
	}

	node := dom.ElementNode("div").WithChildren(dom.TextNode("test"))
	content = nodeToSlotContent(node)
	if len(content.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(content.nodes))
	}
	if content.nodes[0].Tag != "div" {
		t.Errorf("expected div, got %s", content.nodes[0].Tag)
	}

	fragment := dom.FragmentNode()
	fragment.Children = []*dom.StructuredNode{
		dom.ElementNode("h1"),
		dom.ElementNode("p"),
	}
	content = nodeToSlotContent(fragment)
	if len(content.nodes) != 2 {
		t.Errorf("expected 2 nodes from fragment, got %d", len(content.nodes))
	}
}
