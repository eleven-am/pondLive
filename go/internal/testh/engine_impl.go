package testh

import (
	"fmt"
	"strings"

	"github.com/eleven-am/go/pondlive/internal/diff"
	handlers "github.com/eleven-am/go/pondlive/internal/handlers"
	render "github.com/eleven-am/go/pondlive/internal/render"
	"github.com/eleven-am/go/pondlive/pkg/live/html"
)

// NewEngine constructs a harness engine with optional dependencies.
func NewEngine(rec Recorder, factory RegistryFactory, channel ProtocolChannel) Engine {
	if rec == nil {
		rec = NewRecorder()
	}
	if factory == nil {
		factory = NewRegistryFactory()
	}
	if channel == nil {
		channel = NewProtocolChannel()
	}
	return &engine{
		recorder:        rec,
		registryFactory: factory,
		protocol:        channel,
	}
}

type engine struct {
	renderFn func() html.Node

	recorder        Recorder
	registryFactory RegistryFactory
	protocol        ProtocolChannel

	lookup HandlerRegistry
	reg    handlers.Registry

	prev  render.Structured
	dirty bool

	location struct {
		path  string
		query string
	}
}

func (e *engine) Mount(renderFn func() html.Node) {
	e.renderFn = renderFn
	e.lookup = nil
	e.reg = nil
	e.prev = render.Structured{}
	e.dirty = false
	e.location.path = "/"
	e.location.query = ""

	if renderFn == nil {
		e.recorder.SnapshotHTML("")
		e.recorder.SnapshotOps(nil)
		return
	}

	e.lookup = e.registryFactory.NewRegistry()
	reg, ok := e.lookup.(handlers.Registry)
	if !ok {
		panic(fmt.Sprintf("testh: registry %T does not implement handlers.Registry", e.lookup))
	}
	e.reg = reg

	structured, html := e.renderStructuredAndHTML()
	e.prev = structured
	e.recorder.SnapshotHTML(html)
	e.recorder.SnapshotOps(nil)
}

func (e *engine) Flush() {
	if !e.dirty {
		e.recorder.SnapshotOps(nil)
		return
	}
	if e.renderFn == nil {
		e.recorder.SnapshotOps(nil)
		e.dirty = false
		return
	}
	structured, html := e.renderStructuredAndHTML()
	ops := diff.Diff(e.prev, structured)
	e.prev = structured
	e.recorder.SnapshotHTML(html)
	e.recorder.SnapshotOps(ops)
	e.dirty = false
	if e.protocol != nil {
		e.protocol.Enqueue(cloneOps(ops))
	}
}

func (e *engine) HTML() string {
	return e.recorder.HTML()
}

func (e *engine) Structured() render.Structured {
	return cloneStructured(e.prev)
}

func (e *engine) Ops() []diff.Op {
	return e.recorder.Ops()
}

func (e *engine) DispatchEvent(handlerID string, payload map[string]any) {
	fn := e.lookupHandler(handlerID)
	if fn == nil {
		return
	}
	updates := fn(html.Event{ //nolint:exhaustruct
		Name:    inferEventName(handlerID, "event"),
		Payload: clonePayload(payload),
	})
	if updates != nil {
		e.dirty = true
	}
}

func (e *engine) DispatchSubmit(handlerID string, form map[string]string) {
	fn := e.lookupHandler(handlerID)
	if fn == nil {
		return
	}
	updates := fn(html.Event{ //nolint:exhaustruct
		Name: inferEventName(handlerID, "submit"),
		Form: cloneForm(form),
	})
	if updates != nil {
		e.dirty = true
	}
}

func (e *engine) Navigate(path string, query string) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	query = strings.TrimSpace(query)
	if path == e.location.path && query == e.location.query {
		return
	}
	e.location.path = path
	e.location.query = query
	e.dirty = true
}

func (e *engine) ResetOps() {
	e.recorder.ResetOps()
}

func (e *engine) renderStructuredAndHTML() (render.Structured, string) {
	if e.reg == nil {
		e.lookup = e.registryFactory.NewRegistry()
		reg, ok := e.lookup.(handlers.Registry)
		if !ok {
			panic(fmt.Sprintf("testh: registry %T does not implement handlers.Registry", e.lookup))
		}
		e.reg = reg
	}
	node := e.renderFn()
	structured := render.ToStructuredWithHandlers(node, e.reg)
	html := render.RenderHTML(node, e.reg)
	return cloneStructured(structured), html
}

func (e *engine) lookupHandler(handlerID string) html.EventHandler {
	if handlerID == "" || e.lookup == nil {
		return nil
	}
	fn, ok := e.lookup.Lookup(handlerID).(html.EventHandler)
	if !ok {
		return nil
	}
	return fn
}

func inferEventName(handlerID, fallback string) string {
	if handlerID == "" {
		return fallback
	}
	if idx := strings.LastIndex(handlerID, ":"); idx >= 0 && idx < len(handlerID)-1 {
		segment := handlerID[idx+1:]
		if dash := strings.Index(segment, "-"); dash > 0 {
			segment = segment[:dash]
		}
		if segment != "" {
			return segment
		}
	}
	return fallback
}

func clonePayload(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}

func cloneForm(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}
