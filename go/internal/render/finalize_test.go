package render

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestFinalizeWithHandlersMergesMetadata(t *testing.T) {
	reg := handlers.NewRegistry()
	clicked := false
	handler := func(h.Event) h.Updates {
		clicked = true
		return h.Rerender()
	}
	node := h.Div(
		h.Class("p-4", "bg-white"),
		h.Style("color", "red"),
		h.Attr("data-extra", "value"),
		h.On("click", handler),
		h.Span(h.Text("hello")),
	)

	FinalizeWithHandlers(node, reg)

	el := node
	if got, want := el.Attrs["class"], "p-4 bg-white"; got != want {
		t.Fatalf("class merge mismatch: got %q want %q", got, want)
	}
	if got, want := el.Attrs["style"], "color:red;"; got != want {
		t.Fatalf("style merge mismatch: got %q want %q", got, want)
	}
	if got := el.Attrs["data-extra"]; got != "value" {
		t.Fatalf("expected data attribute, got %q", got)
	}
	if id := el.Attrs["data-onclick"]; id == "" {
		t.Fatal("expected data-onclick attribute to be set")
	}
	if clicked {
		t.Fatal("handler should not have been executed during finalize")
	}
}

func TestRenderHTMLWithHandlersIncludesEventData(t *testing.T) {
	reg := handlers.NewRegistry()
	handler := func(h.Event) h.Updates { return h.Rerender() }
	html := RenderHTML(h.Button(h.On("click", handler), h.Text("hi")), reg)
	if !strings.Contains(html, "data-onclick") {
		t.Fatalf("expected rendered html to include data-onclick, got %q", html)
	}
}

func TestFinalizeAnnotatesRowKeys(t *testing.T) {
	node := h.Li(h.Key("item-1"), h.Text("row"))
	Finalize(node)
	if node.Attrs["data-row-key"] != "item-1" {
		t.Fatalf("expected data-row-key to be set, got %q", node.Attrs["data-row-key"])
	}
}
