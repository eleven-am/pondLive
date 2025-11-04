package runtime

import (
	"github.com/eleven-am/go/pondlive/internal/protocol"
	render "github.com/eleven-am/go/pondlive/internal/render"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
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
