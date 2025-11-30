package work

import "github.com/eleven-am/pondlive/go/internal/metadata"

type Modifiers struct {
	Ctrl   bool
	Meta   bool
	Shift  bool
	Alt    bool
	Button int
}

type Event struct {
	Name    string
	Value   string
	Payload map[string]any
	Form    map[string]string
	Mods    Modifiers
}

type Updates any

type Handler struct {
	metadata.EventOptions
	Fn func(Event) Updates
}
