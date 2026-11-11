package work

import "github.com/eleven-am/pondlive/internal/metadata"

type Event struct {
	Name    string
	Payload map[string]any
}

type Handler struct {
	metadata.EventOptions
	Fn func(Event)
}

type EventBinding struct {
	Event   string
	Options metadata.EventOptions
	Fns     []func(Event)
}
