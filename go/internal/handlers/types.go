package handlers

import h "github.com/eleven-am/pondlive/go/pkg/live/html"

type ID string

type Event = h.Event

type Modifiers = h.Modifiers

type Updates = h.Updates

type Handler = h.EventHandler

// Registry tracks event handlers and assigns stable IDs.
type Registry interface {
	Ensure(fn Handler) ID
	Get(id ID) (Handler, bool)
	Remove(id ID)
}
