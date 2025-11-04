package testh

import (
	"github.com/eleven-am/go/pondlive/internal/diff"
	"github.com/eleven-am/go/pondlive/internal/render"
	"github.com/eleven-am/go/pondlive/pkg/live/html"
)

// Engine encapsulates a single in-memory LiveUI session under test.
type Engine interface {
	// Boot with a render function. Produces initial Structured and SSR HTML.
	Mount(render func() html.Node)

	// Force a flush if dirty; record ops; update previous Structured.
	Flush()

	// Readbacks
	HTML() string
	Structured() render.Structured
	Ops() []diff.Op

	// Simulations
	DispatchEvent(handlerID string, payload map[string]any)
	DispatchSubmit(handlerID string, form map[string]string)
	Navigate(path string, query string)

	// Maintenance
	ResetOps()
}
