package render

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestRouterPathMustStartWithSlash(t *testing.T) {
	button := h.Button(h.Text("click"))
	button.Attrs = map[string]string{
		"data-router-path": "relative/path",
	}
	root := h.WrapComponent("root", h.Div(button))

	_, err := ToStructured(root)
	if err == nil {
		t.Fatalf("expected error for router path not starting with /, got nil")
	}
}

func TestRouterBindingEmptyAttributesAreIgnored(t *testing.T) {
	button := h.Button(h.Text("noop"))
	button.Attrs = map[string]string{
		"data-router-path":    "",
		"data-router-query":   "",
		"data-router-hash":    "",
		"data-router-replace": "",
	}
	root := h.WrapComponent("root", h.Div(button))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.RouterBindings) != 0 {
		t.Fatalf("expected no router bindings when attributes are empty, got %d", len(structured.RouterBindings))
	}
}
