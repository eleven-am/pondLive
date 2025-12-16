package view

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
)

func TestRenderHTML_Nil(t *testing.T) {
	result := RenderHTML(nil)
	if result != "" {
		t.Errorf("expected empty string for nil, got %q", result)
	}
}

func TestRenderHTML_Text(t *testing.T) {
	node := &Text{Text: "Hello, World!"}
	result := RenderHTML(node)
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}
}

func TestRenderHTML_TextEscaped(t *testing.T) {
	node := &Text{Text: "<script>alert('xss')</script>"}
	result := RenderHTML(node)
	if strings.Contains(result, "<script>") {
		t.Error("expected HTML to be escaped")
	}
}

func TestRenderHTML_Comment(t *testing.T) {
	node := &Comment{Comment: "This is a comment"}
	result := RenderHTML(node)
	if result != "<!--This is a comment-->" {
		t.Errorf("expected comment syntax, got %q", result)
	}
}

func TestRenderHTML_CommentEscaped(t *testing.T) {
	node := &Comment{Comment: "foo --> bar"}
	result := RenderHTML(node)
	if !strings.Contains(result, "&gt;") {
		t.Error("expected > to be escaped in comment")
	}
}

func TestRenderHTML_Element_Simple(t *testing.T) {
	node := &Element{Tag: "div"}
	result := RenderHTML(node)
	if result != "<div></div>" {
		t.Errorf("expected '<div></div>', got %q", result)
	}
}

func TestRenderHTML_Element_WithAttrs(t *testing.T) {
	node := &Element{
		Tag: "div",
		Attrs: map[string][]string{
			"class": {"container", "main"},
			"id":    {"app"},
		},
	}
	result := RenderHTML(node)
	if !strings.Contains(result, `class="container main"`) {
		t.Errorf("expected class attr, got %q", result)
	}
	if !strings.Contains(result, `id="app"`) {
		t.Errorf("expected id attr, got %q", result)
	}
}

func TestRenderHTML_Element_BooleanAttr(t *testing.T) {
	node := &Element{
		Tag: "input",
		Attrs: map[string][]string{
			"type":     {"checkbox"},
			"checked":  {},
			"disabled": {},
		},
	}
	result := RenderHTML(node)
	if !strings.Contains(result, " checked") {
		t.Errorf("expected checked boolean attr, got %q", result)
	}
	if !strings.Contains(result, " disabled") {
		t.Errorf("expected disabled boolean attr, got %q", result)
	}
}

func TestRenderHTML_Element_WithStyle(t *testing.T) {
	node := &Element{
		Tag: "div",
		Style: map[string]string{
			"color":      "red",
			"background": "blue",
		},
	}
	result := RenderHTML(node)
	if !strings.Contains(result, `style="`) {
		t.Errorf("expected style attr, got %q", result)
	}
	if !strings.Contains(result, "color: red") {
		t.Errorf("expected color style, got %q", result)
	}
}

func TestRenderHTML_Element_WithChildren(t *testing.T) {
	node := &Element{
		Tag: "div",
		Children: []Node{
			&Text{Text: "Hello"},
			&Element{Tag: "span", Children: []Node{&Text{Text: "World"}}},
		},
	}
	result := RenderHTML(node)
	if result != "<div>Hello<span>World</span></div>" {
		t.Errorf("expected nested HTML, got %q", result)
	}
}

func TestRenderHTML_Element_UnsafeHTML(t *testing.T) {
	node := &Element{
		Tag:        "div",
		UnsafeHTML: "<b>Bold</b>",
	}
	result := RenderHTML(node)
	if !strings.Contains(result, "<b>Bold</b>") {
		t.Errorf("expected unsafe HTML to be rendered, got %q", result)
	}
}

func TestRenderHTML_Element_HTML_Doctype(t *testing.T) {
	node := &Element{
		Tag: "html",
		Children: []Node{
			&Element{Tag: "head"},
			&Element{Tag: "body"},
		},
	}
	result := RenderHTML(node)
	if !strings.HasPrefix(result, "<!DOCTYPE html>") {
		t.Errorf("expected DOCTYPE, got %q", result)
	}
}

func TestRenderHTML_VoidElements(t *testing.T) {
	voidTags := []string{"br", "hr", "img", "input", "meta", "link"}
	for _, tag := range voidTags {
		node := &Element{Tag: tag}
		result := RenderHTML(node)
		if strings.Contains(result, "</"+tag+">") {
			t.Errorf("void element %s should not have closing tag, got %q", tag, result)
		}
	}
}

func TestRenderHTML_Fragment(t *testing.T) {
	node := &Fragment{
		Children: []Node{
			&Text{Text: "One"},
			&Text{Text: "Two"},
		},
	}
	result := RenderHTML(node)
	if result != "OneTwo" {
		t.Errorf("expected 'OneTwo', got %q", result)
	}
}

func TestRenderHTML_Fragment_Empty(t *testing.T) {
	node := &Fragment{Children: nil}
	result := RenderHTML(node)
	if result != "" {
		t.Errorf("expected empty string for empty fragment, got %q", result)
	}
}

func TestRenderHTML_Element_WithStylesheet(t *testing.T) {
	node := &Element{
		Tag: "style",
		Stylesheet: &metadata.Stylesheet{
			Rules: []metadata.StyleRule{
				{
					Selector: ".test",
					Decls:    []metadata.Declaration{{Property: "color", Value: "red"}},
				},
			},
		},
	}
	result := RenderHTML(node)
	if !strings.Contains(result, ".test{color:red}") {
		t.Errorf("expected CSS rule, got %q", result)
	}
}

func TestRenderHTML_Element_WithStylesheet_MediaQueries(t *testing.T) {
	node := &Element{
		Tag: "style",
		Stylesheet: &metadata.Stylesheet{
			MediaBlocks: []metadata.MediaBlock{
				{
					Query: "(max-width: 768px)",
					Rules: []metadata.StyleRule{
						{
							Selector: ".mobile",
							Decls:    []metadata.Declaration{{Property: "display", Value: "block"}},
						},
					},
				},
			},
		},
	}
	result := RenderHTML(node)
	if !strings.Contains(result, "@media (max-width: 768px)") {
		t.Errorf("expected media query, got %q", result)
	}
	if !strings.Contains(result, ".mobile{display:block}") {
		t.Errorf("expected rule inside media query, got %q", result)
	}
}

func TestEscapeCSSForHTML(t *testing.T) {
	input := "content: '</script>'"
	result := escapeCSSForHTML(input)
	if strings.Contains(result, "</script>") {
		t.Error("expected </script> to be escaped")
	}
}

func TestWriteDecls_Empty(t *testing.T) {
	var b strings.Builder
	writeDecls(&b, nil)
	if b.Len() != 0 {
		t.Errorf("expected empty output for nil decls, got %q", b.String())
	}
}

func TestWriteDecls_Multiple(t *testing.T) {
	var b strings.Builder
	writeDecls(&b, []metadata.Declaration{
		{Property: "a", Value: "1"},
		{Property: "b", Value: "2"},
	})
	result := b.String()
	if result != "a:1;b:2" {
		t.Errorf("expected a:1;b:2, got %q", result)
	}
}

func TestRenderStylesheet_Nil(t *testing.T) {
	var b strings.Builder
	renderStylesheet(&b, nil)
	if b.Len() != 0 {
		t.Errorf("expected empty output for nil stylesheet, got %q", b.String())
	}
}

func TestViewNode_Methods(t *testing.T) {
	e := &Element{}
	e.viewNode()

	txt := &Text{}
	txt.viewNode()

	c := &Comment{}
	c.viewNode()

	f := &Fragment{}
	f.viewNode()
}

func TestRenderHTML_AttrEscaping(t *testing.T) {
	node := &Element{
		Tag: "div",
		Attrs: map[string][]string{
			"data-value": {`"quoted"`},
		},
	}
	result := RenderHTML(node)
	if strings.Contains(result, `""`) {
		t.Error("expected quotes to be escaped in attribute value")
	}
}
