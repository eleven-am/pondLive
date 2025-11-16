package router

import (
	"strings"

	h "github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/render"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

// Test helper functions for HTML element creation

func testDiv(items ...h.Item) h.Node {
	return h.El(h.HTMLDivElement{}, "div", items...)
}

func testButton(items ...h.Item) h.Node {
	return h.El(h.HTMLButtonElement{}, "button", items...)
}

func testSpan(items ...h.Item) h.Node {
	return h.El(h.HTMLSpanElement{}, "span", items...)
}

func testH1(items ...h.Item) h.Node {
	return h.El(h.HTMLH1Element{}, "h1", items...)
}

func testNav(items ...h.Item) h.Node {
	return h.El(h.HTMLNavElement{}, "nav", items...)
}

// Component type alias for tests
type Component[P any] = runtime.Component[P]

// Type alias for Meta
type Meta = runtime.Meta

// Additional helper functions for tests
func findHandlerAttr(structured render.Structured, attr string) string {
	event := strings.TrimPrefix(attr, "data-on")
	if idx := strings.IndexByte(event, '-'); idx != -1 {
		event = event[:idx]
	}
	event = strings.TrimSpace(event)
	if event == "" {
		return ""
	}
	for _, binding := range structured.Bindings {
		if binding.Event == event && binding.Handler != "" {
			return binding.Handler
		}
	}
	return ""
}

func findClickHandlerID(structured render.Structured) string {
	return findHandlerAttr(structured, "data-onclick")
}
