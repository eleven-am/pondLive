package dom

import (
	"testing"
)

func TestFlattenNil(t *testing.T) {
	var node *StructuredNode
	result := node.Flatten()
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestFlattenTextNode(t *testing.T) {
	node := &StructuredNode{Text: "hello"}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Text != "hello" {
		t.Errorf("expected text 'hello', got %q", result.Text)
	}
}

func TestFlattenCommentNode(t *testing.T) {
	node := &StructuredNode{Comment: "a comment"}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Comment != "a comment" {
		t.Errorf("expected comment 'a comment', got %q", result.Comment)
	}
}

func TestFlattenElementNode(t *testing.T) {
	node := &StructuredNode{
		Tag:   "div",
		Key:   "my-key",
		RefID: "ref-1",
		Attrs: map[string][]string{"class": {"foo", "bar"}},
		Style: map[string]string{"color": "red"},
		Children: []*StructuredNode{
			{Text: "child1"},
			{Text: "child2"},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Tag != "div" {
		t.Errorf("expected tag 'div', got %q", result.Tag)
	}
	if result.Key != "my-key" {
		t.Errorf("expected key 'my-key', got %q", result.Key)
	}
	if result.RefID != "ref-1" {
		t.Errorf("expected refID 'ref-1', got %q", result.RefID)
	}
	if len(result.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(result.Children))
	}
}

func TestFlattenFragmentSingleChild(t *testing.T) {
	node := &StructuredNode{
		Fragment: true,
		Children: []*StructuredNode{
			{Tag: "span", Children: []*StructuredNode{{Text: "inner"}}},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Fragment {
		t.Error("single-child fragment should not be a fragment")
	}
	if result.Tag != "span" {
		t.Errorf("expected tag 'span', got %q", result.Tag)
	}
}

func TestFlattenFragmentMultipleChildren(t *testing.T) {
	node := &StructuredNode{
		Fragment: true,
		Children: []*StructuredNode{
			{Tag: "span"},
			{Tag: "div"},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Fragment {
		t.Error("multiple-child fragment should remain a fragment")
	}
	if len(result.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(result.Children))
	}
}

func TestFlattenFragmentEmpty(t *testing.T) {
	node := &StructuredNode{
		Fragment: true,
		Children: nil,
	}
	result := node.Flatten()

	if result != nil {
		t.Errorf("empty fragment should flatten to nil, got %+v", result)
	}
}

func TestFlattenComponent(t *testing.T) {
	node := &StructuredNode{
		ComponentID: "my-component",
		Children: []*StructuredNode{
			{Tag: "div", Children: []*StructuredNode{{Text: "content"}}},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ComponentID != "" {
		t.Error("component ID should be removed")
	}
	if result.Tag != "div" {
		t.Errorf("expected tag 'div', got %q", result.Tag)
	}
}

func TestFlattenNestedFragments(t *testing.T) {
	node := &StructuredNode{
		Tag: "div",
		Children: []*StructuredNode{
			{Fragment: true, Children: []*StructuredNode{
				{Text: "a"},
				{Fragment: true, Children: []*StructuredNode{
					{Text: "b"},
					{Text: "c"},
				}},
				{Text: "d"},
			}},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Tag != "div" {
		t.Errorf("expected tag 'div', got %q", result.Tag)
	}

	if len(result.Children) != 4 {
		t.Errorf("expected 4 children after flattening, got %d", len(result.Children))
	}
	expected := []string{"a", "b", "c", "d"}
	for i, child := range result.Children {
		if child.Text != expected[i] {
			t.Errorf("child %d: expected text %q, got %q", i, expected[i], child.Text)
		}
	}
}

func TestFlattenNestedComponents(t *testing.T) {
	node := &StructuredNode{
		ComponentID: "outer",
		Children: []*StructuredNode{
			{ComponentID: "inner", Children: []*StructuredNode{
				{Tag: "button", Children: []*StructuredNode{{Text: "Click"}}},
			}},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ComponentID != "" {
		t.Error("component ID should be removed")
	}
	if result.Tag != "button" {
		t.Errorf("expected tag 'button', got %q", result.Tag)
	}
	if len(result.Children) != 1 || result.Children[0].Text != "Click" {
		t.Error("children not preserved correctly")
	}
}

func TestFlattenPreservesHandlers(t *testing.T) {
	handlers := []HandlerMeta{
		{Event: "click", Handler: "h1"},
	}
	node := &StructuredNode{
		Tag:      "button",
		Handlers: handlers,
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(result.Handlers))
	}
	if result.Handlers[0].Event != "click" {
		t.Error("handler not preserved")
	}
}

func TestFlattenPreservesRouter(t *testing.T) {
	node := &StructuredNode{
		Tag: "a",
		Router: &RouterMeta{
			PathValue: "/home",
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Router == nil || result.Router.PathValue != "/home" {
		t.Error("router not preserved")
	}
}

func TestFlattenPreservesUpload(t *testing.T) {
	node := &StructuredNode{
		Tag: "input",
		Upload: &UploadMeta{
			UploadID: "up-1",
			Multiple: true,
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Upload == nil || result.Upload.UploadID != "up-1" {
		t.Error("upload not preserved")
	}
}

func TestFlattenMixedComponentsAndElements(t *testing.T) {
	node := &StructuredNode{
		ComponentID: "App",
		Children: []*StructuredNode{
			{Tag: "header", Children: []*StructuredNode{{Text: "Header"}}},
			{ComponentID: "Content", Children: []*StructuredNode{
				{Tag: "main", Children: []*StructuredNode{
					{Fragment: true, Children: []*StructuredNode{
						{Tag: "p", Children: []*StructuredNode{{Text: "Para 1"}}},
						{Tag: "p", Children: []*StructuredNode{{Text: "Para 2"}}},
					}},
				}},
			}},
			{Tag: "footer", Children: []*StructuredNode{{Text: "Footer"}}},
		},
	}
	result := node.Flatten()

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.Fragment {
		t.Error("expected result to be a fragment")
	}
	if len(result.Children) != 3 {
		t.Errorf("expected 3 top-level children, got %d", len(result.Children))
	}

	if result.Children[0].Tag != "header" {
		t.Errorf("expected first child to be header, got %q", result.Children[0].Tag)
	}
	if result.Children[1].Tag != "main" {
		t.Errorf("expected second child to be main, got %q", result.Children[1].Tag)
	}
	if result.Children[2].Tag != "footer" {
		t.Errorf("expected third child to be footer, got %q", result.Children[2].Tag)
	}

	mainNode := result.Children[1]
	if len(mainNode.Children) != 2 {
		t.Errorf("expected main to have 2 children after flattening, got %d", len(mainNode.Children))
	}
}
