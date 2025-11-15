package render

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestRenderHTMLBasicElement(t *testing.T) {
	node := h.Div(h.Text("hello"))
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if html != "<div>hello</div>" {
		t.Errorf("expected '<div>hello</div>', got %q", html)
	}
}

func TestRenderHTMLNestedElements(t *testing.T) {
	node := h.Div(
		h.Span(h.Text("first")),
		h.Span(h.Text("second")),
	)
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	expected := "<div><span>first</span><span>second</span></div>"
	if html != expected {
		t.Errorf("expected %q, got %q", expected, html)
	}
}

func TestRenderHTMLWithAttributes(t *testing.T) {
	node := h.Div(
		h.Attr("class", "container"),
		h.Attr("id", "main"),
		h.Text("content"),
	)
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if !strings.Contains(html, `class="container"`) {
		t.Errorf("expected class attribute, got %q", html)
	}
	if !strings.Contains(html, `id="main"`) {
		t.Errorf("expected id attribute, got %q", html)
	}
}

func TestRenderHTMLComponent(t *testing.T) {
	inner := h.P(h.Text("component content"))
	node := h.WrapComponent("test-comp", inner)
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	expected := "<p>component content</p>"
	if html != expected {
		t.Errorf("expected %q, got %q", expected, html)
	}
}

func TestRenderHTMLFragment(t *testing.T) {
	node := h.Fragment(
		h.Div(h.Text("first")),
		h.Div(h.Text("second")),
	)
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	expected := "<div>first</div><div>second</div>"
	if html != expected {
		t.Errorf("expected %q, got %q", expected, html)
	}
}

func TestRenderHTMLVoidElement(t *testing.T) {
	node := h.Input(h.Attr("type", "text"), h.Attr("value", "test"))
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if strings.Contains(html, "</input>") {
		t.Errorf("void element should not have closing tag, got %q", html)
	}
	if !strings.HasPrefix(html, "<input") || !strings.HasSuffix(html, ">") {
		t.Errorf("expected void element format, got %q", html)
	}
}

func TestRenderHTMLEscapesText(t *testing.T) {
	node := h.Div(h.Text("<script>alert('xss')</script>"))
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if strings.Contains(html, "<script>") {
		t.Errorf("text should be escaped, got %q", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("expected escaped content, got %q", html)
	}
}

func TestRenderHTMLEscapesAttributes(t *testing.T) {
	node := h.Div(h.Attr("title", `"quoted" & special`))
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	hasQuoteEscaped := strings.Contains(html, "&quot;") || strings.Contains(html, "&#34;")
	hasAmpEscaped := strings.Contains(html, "&amp;")
	if !hasQuoteEscaped || !hasAmpEscaped {
		t.Errorf("attributes should be escaped, got %q", html)
	}
}

func TestRenderHTMLUnsafeContent(t *testing.T) {
	unsafeContent := "<strong>raw html</strong>"
	elem := h.Div()
	elem.Unsafe = &unsafeContent
	reg := handlers.NewRegistry()
	html := RenderHTML(elem, reg)

	if !strings.Contains(html, "<strong>raw html</strong>") {
		t.Errorf("unsafe content should be raw, got %q", html)
	}
}

func TestRenderHTMLComment(t *testing.T) {
	node := h.Comment("test comment")
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	expected := "<!--test comment-->"
	if html != expected {
		t.Errorf("expected %q, got %q", expected, html)
	}
}

func TestRenderHTMLCommentEscapesDoubleDash(t *testing.T) {
	node := h.Comment("test -- comment")
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if strings.Contains(html, "--") && !strings.HasPrefix(html, "<!--") {
		t.Errorf("double dash should be escaped in comment, got %q", html)
	}
	expected := "<!--test - - comment-->"
	if html != expected {
		t.Errorf("expected %q, got %q", expected, html)
	}
}

func TestRenderHTMLNilNode(t *testing.T) {
	reg := handlers.NewRegistry()
	html := RenderHTML(nil, reg)

	if html != "" {
		t.Errorf("expected empty string for nil node, got %q", html)
	}
}

func TestRenderHTMLEmptyAttributes(t *testing.T) {
	node := h.Div(
		h.Attr("class", "test"),
		h.Attr("data-empty", ""),
		h.Text("content"),
	)
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if strings.Contains(html, `data-empty=""`) {
		t.Errorf("empty attributes should be skipped, got %q", html)
	}
	if !strings.Contains(html, `class="test"`) {
		t.Errorf("non-empty attributes should remain, got %q", html)
	}
}

func TestRenderHTMLComplexStructure(t *testing.T) {
	node := h.WrapComponent("root",
		h.Div(
			h.Attr("class", "container"),
			h.H1(h.Text("Title")),
			h.Ul(
				h.Li(h.Text("Item 1")),
				h.Li(h.Text("Item 2")),
				h.Li(h.Text("Item 3")),
			),
			h.P(h.Text("Footer text")),
		),
	)
	reg := handlers.NewRegistry()
	html := RenderHTML(node, reg)

	if !strings.Contains(html, "<h1>Title</h1>") {
		t.Errorf("missing h1 element, got %q", html)
	}
	if !strings.Contains(html, "<ul>") || !strings.Contains(html, "</ul>") {
		t.Errorf("missing ul element, got %q", html)
	}
	if strings.Count(html, "<li>") != 3 {
		t.Errorf("expected 3 li elements, got %q", html)
	}
}
