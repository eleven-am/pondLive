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
	assignment, ok := el.HandlerAssignments["click"]
	if !ok || assignment.ID == "" {
		t.Fatal("expected click handler assignment to be recorded")
	}
	if clicked {
		t.Fatal("handler should not have been executed during finalize")
	}
}

func TestRenderHTMLWithHandlersOmitsInlineEventData(t *testing.T) {
	reg := handlers.NewRegistry()
	handler := func(h.Event) h.Updates { return h.Rerender() }
	button := h.Button(h.On("click", handler), h.Text("hi"))
	html := RenderHTML(button, reg)
	if strings.Contains(html, "data-onclick") {
		t.Fatalf("expected rendered html to omit data-onclick, got %q", html)
	}
	structured := ToStructuredWithHandlers(h.Button(h.On("click", handler), h.Text("hi")), reg)
	if len(structured.Bindings) == 0 {
		t.Fatalf("expected handler bindings to be recorded, got none")
	}
	var found bool
	for _, binding := range structured.Bindings {
		if binding.Event == "click" && binding.Handler != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected click binding to be recorded, got %+v", structured.Bindings)
	}
}

func TestFinalizeWithHandlersAppliesEventMetadata(t *testing.T) {
	reg := handlers.NewRegistry()
	handler := func(h.Event) h.Updates { return h.Rerender() }
	node := h.Video(
		h.OnWith("timeupdate", h.EventOptions{
			Listen: []string{"play", "pause", "timeupdate"},
			Props:  []string{"target.currentTime", "target.duration"},
		}, handler),
	)

	FinalizeWithHandlers(node, reg)

	assignment, ok := node.HandlerAssignments["timeupdate"]
	if !ok || assignment.ID == "" {
		t.Fatal("expected handler assignment for primary event")
	}
	if got := strings.Join(assignment.Listen, " "); got != "play pause" {
		t.Fatalf("unexpected listen metadata: %q", got)
	}
	if got := strings.Join(assignment.Props, " "); got != "target.currentTime target.duration target.paused" {
		t.Fatalf("unexpected props metadata: %q", got)
	}
}

func TestFinalizeAnnotatesRowKeys(t *testing.T) {
	node := h.Li(h.Key("item-1"), h.Text("row"))
	Finalize(node)
	if node.Attrs["data-row-key"] != "item-1" {
		t.Fatalf("expected data-row-key to be set, got %q", node.Attrs["data-row-key"])
	}
}
