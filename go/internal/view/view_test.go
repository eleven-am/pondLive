package view

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/metadata"
)

func TestElementCreation(t *testing.T) {
	el := &Element{
		Tag: "div",
		Attrs: map[string][]string{
			"id":    {"test"},
			"class": {"foo", "bar"},
		},
		Style: map[string]string{
			"color": "red",
		},
	}

	if el.Tag != "div" {
		t.Errorf("expected tag='div', got %s", el.Tag)
	}
	if el.Attrs["id"][0] != "test" {
		t.Error("id attribute not set correctly")
	}
	if len(el.Attrs["class"]) != 2 {
		t.Errorf("expected 2 classes, got %d", len(el.Attrs["class"]))
	}
	if el.Style["color"] != "red" {
		t.Error("style not set correctly")
	}
}

func TestElementWithChildren(t *testing.T) {
	parent := &Element{Tag: "div"}
	child1 := &Element{Tag: "span"}
	child2 := &Text{Text: "hello"}
	child3 := &Comment{Comment: "comment"}

	parent.Children = []Node{child1, child2, child3}

	if len(parent.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(parent.Children))
	}
}

func TestElementWithHandlers(t *testing.T) {
	el := &Element{
		Tag: "button",
		Handlers: []metadata.HandlerMeta{
			{
				Event:   "click",
				Handler: "h1",
			},
			{
				Event:   "mouseover",
				Handler: "h2",
			},
		},
	}

	if len(el.Handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(el.Handlers))
	}
	if el.Handlers[0].Event != "click" {
		t.Error("first handler event incorrect")
	}
	if el.Handlers[0].Handler != "h1" {
		t.Error("first handler ID incorrect")
	}
}

func TestElementWithScript(t *testing.T) {
	el := &Element{
		Tag: "div",
		Script: &metadata.ScriptMeta{
			ScriptID: "script-1",
			Script:   "console.log('test')",
		},
	}

	if el.Script == nil {
		t.Fatal("script metadata not set")
	}
	if el.Script.ScriptID != "script-1" {
		t.Error("script ID incorrect")
	}
}

func TestElementWithStylesheet(t *testing.T) {
	el := &Element{
		Tag: "style",
		Stylesheet: &metadata.Stylesheet{
			Hash: "abc123",
			Rules: []metadata.StyleRule{
				{
					Selector: ".card",
					Props: map[string]string{
						"padding": "16px",
					},
				},
			},
		},
	}

	if el.Stylesheet == nil {
		t.Fatal("stylesheet not set")
	}
	if el.Stylesheet.Hash != "abc123" {
		t.Error("stylesheet hash incorrect")
	}
	if len(el.Stylesheet.Rules) != 1 {
		t.Error("stylesheet rules not set")
	}
}

func TestElementWithUnsafeHTML(t *testing.T) {
	el := &Element{
		Tag:        "div",
		UnsafeHTML: "<span>raw html</span>",
	}

	if el.UnsafeHTML != "<span>raw html</span>" {
		t.Error("unsafe HTML not set correctly")
	}
}

func TestTextNode(t *testing.T) {
	text := &Text{Text: "Hello World"}

	if text.Text != "Hello World" {
		t.Errorf("expected 'Hello World', got %s", text.Text)
	}
}

func TestCommentNode(t *testing.T) {
	comment := &Comment{Comment: "TODO: fix this"}

	if comment.Comment != "TODO: fix this" {
		t.Errorf("expected comment value, got %s", comment.Comment)
	}
}

func TestFragmentNode(t *testing.T) {
	fragment := &Fragment{
		Fragment: true,
		Children: []Node{
			&Text{Text: "hello"},
			&Element{Tag: "span"},
			&Text{Text: "world"},
		},
	}

	if len(fragment.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(fragment.Children))
	}
}

func TestElementWithKey(t *testing.T) {
	el := &Element{
		Tag: "div",
		Key: "user-123",
	}

	if el.Key != "user-123" {
		t.Errorf("expected key='user-123', got %s", el.Key)
	}
}

func TestElementWithRefID(t *testing.T) {
	el := &Element{
		Tag:   "button",
		RefID: "ref-456",
	}

	if el.RefID != "ref-456" {
		t.Errorf("expected refID='ref-456', got %s", el.RefID)
	}
}

func TestNestedStructure(t *testing.T) {
	root := &Element{
		Tag: "div",
		Attrs: map[string][]string{
			"class": {"container"},
		},
		Children: []Node{
			&Element{
				Tag: "header",
				Children: []Node{
					&Text{Text: "Title"},
				},
			},
			&Element{
				Tag: "main",
				Children: []Node{
					&Element{
						Tag: "article",
						Children: []Node{
							&Text{Text: "Content"},
						},
					},
				},
			},
			&Element{
				Tag: "footer",
				Children: []Node{
					&Comment{Comment: "End of page"},
				},
			},
		},
	}

	if len(root.Children) != 3 {
		t.Errorf("expected 3 top-level children, got %d", len(root.Children))
	}

	header := root.Children[0].(*Element)
	if header.Tag != "header" {
		t.Error("first child should be header")
	}

	main := root.Children[1].(*Element)
	if main.Tag != "main" {
		t.Error("second child should be main")
	}

	footer := root.Children[2].(*Element)
	if footer.Tag != "footer" {
		t.Error("third child should be footer")
	}
}
