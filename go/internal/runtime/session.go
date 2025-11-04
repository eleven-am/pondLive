package runtime

import (
	"github.com/eleven-am/liveui/internal/protocol"
	render "github.com/eleven-am/liveui/internal/render"
	h "github.com/eleven-am/liveui/pkg/liveui/html"
)

type SessionID string

type Session interface {
	ID() SessionID
	Version() int
	SetVersion(v int)

	RenderRoot() h.Node

	Prev() render.Structured
	SetPrev(render.Structured)

	Registry() HandlerRegistry

	MarkDirty()
	Dirty() bool

	SetLocation(path string, query string) (changed bool)
	Location() Location

	SendFrame(protocol.Frame) error

	Flush() error
}

type Location struct {
	Path   string
	Query  string
	Params map[string]string
}
