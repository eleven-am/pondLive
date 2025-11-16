package test

import (
	"sort"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/render"
	"github.com/eleven-am/pondlive/go/internal/testh"
	"github.com/eleven-am/pondlive/go/pkg/live/html"
)

// Harness is the public test driver. All methods are synchronous.
// They must be safe to call from a single goroutine (no concurrency required).
type Harness interface {
	// Mount sets up an in-memory LiveUI session with the provided root render function.
	// It performs the initial render and stores the initial (S,D).
	Mount(render func() html.Node)

	// HTML returns the current SSR HTML string for snapshot-style testing.
	// It should reflect the final post-flush state, if Flush() has been called.
	HTML() string

	// SD returns the latest structured render (statics, dynamics).
	// Dynamics should be in a JSON-safe representation (opaque 'any' is acceptable).
	SD() (statics []string, dynamics any)

	// Ops returns the diff ops emitted by the most recent Flush().
	// If no flush has occurred since the last call, return the cached set (idempotent read).
	Ops() []diff.Op

	// Click simulates invoking a server-side event handler by ID (hid),
	// passing an optional payload. It must run the handler and schedule a flush.
	// Implementation detail: in production, the client sends {t:"evt", hid, payload}.
	Click(handlerID string, payload map[string]any)

	// Submit is a convenience for common form submits; equivalent to Click with
	// an Event containing a Form map (implementation-defined).
	Submit(handlerID string, form map[string]string)

	// Nav simulates navigation: sets Location (path, query) and schedules a flush,
	// as if the client sent {t:"nav", path, q}.
	Nav(path string, query string)

	// Flush forces the scheduler to render, diff, and record ops if the session is dirty.
	// If not dirty, Flush must be a no-op and record zero ops.
	Flush()

	// ResetOps clears the recorded ops buffer without altering state (for step-by-step assertions).
	ResetOps()
}

// NewHarness constructs a new Harness with default fakes (no external I/O).
func NewHarness() Harness {
	return &harness{engine: testh.NewEngine(nil, nil, nil)}
}

type harness struct {
	engine testh.Engine
}

func (h *harness) Mount(render func() html.Node) {
	if h == nil {
		return
	}
	h.engine.Mount(render)
}

func (h *harness) HTML() string {
	if h == nil {
		return ""
	}
	return h.engine.HTML()
}

func (h *harness) SD() ([]string, any) {
	if h == nil {
		return nil, nil
	}
	structured := h.engine.Structured()
	statics := append([]string(nil), structured.S...)
	dynamics := encodeDynamics(structured.D)
	return statics, dynamics
}

func (h *harness) Ops() []diff.Op {
	if h == nil {
		return nil
	}
	return h.engine.Ops()
}

func (h *harness) Click(handlerID string, payload map[string]any) {
	if h == nil {
		return
	}
	h.engine.DispatchEvent(handlerID, payload)
}

func (h *harness) Submit(handlerID string, form map[string]string) {
	if h == nil {
		return
	}
	h.engine.DispatchSubmit(handlerID, form)
}

func (h *harness) Nav(path string, query string) {
	if h == nil {
		return
	}
	h.engine.Navigate(path, query)
}

func (h *harness) Flush() {
	if h == nil {
		return
	}
	h.engine.Flush()
}

func (h *harness) ResetOps() {
	if h == nil {
		return
	}
	h.engine.ResetOps()
}

func encodeDynamics(dyns []render.DynamicSlot) any {
	if len(dyns) == 0 {
		return []any{}
	}
	out := make([]map[string]any, 0, len(dyns))
	for _, dyn := range dyns {
		slot := map[string]any{}
		switch dyn.Kind {
		case render.DynamicText:
			slot["k"] = "text"
			slot["text"] = dyn.Text
		case render.DynamicAttrs:
			slot["k"] = "attrs"
			attrs := make([]map[string]string, 0, len(dyn.Attrs))
			if len(dyn.Attrs) > 0 {
				keys := make([]string, 0, len(dyn.Attrs))
				for k := range dyn.Attrs {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					attrs = append(attrs, map[string]string{
						"name":  k,
						"value": dyn.Attrs[k],
					})
				}
			}
			slot["attrs"] = attrs
		case render.DynamicList:
			slot["k"] = "list"
			rows := make([]map[string]any, 0, len(dyn.List))
			for _, row := range dyn.List {
				rows = append(rows, map[string]any{
					"key":   row.Key,
					"slots": append([]int(nil), row.Slots...),
				})
			}
			slot["rows"] = rows
		}
		out = append(out, slot)
	}
	return out
}
