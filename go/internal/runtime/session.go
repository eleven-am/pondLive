package runtime

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	render "github.com/eleven-am/pondlive/go/internal/render"
)

type SessionID string

type Session interface {
	ID() SessionID
	Version() int
	SetVersion(v int)

	RenderRoot() dom.Node

	Prev() render.Structured
	SetPrev(render.Structured)

	Registry() HandlerRegistry

	MarkDirty()
	Dirty() bool

	SetLocation(path string, query string) (changed bool)
	Location() SessionLocation

	SendFrame(protocol.Frame) error

	Flush() error
}

type SessionLocation struct {
	Path   string
	Query  string
	Params map[string]string
}
