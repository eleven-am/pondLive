package work

import "github.com/eleven-am/pondlive/go/internal/metadata"

// Modifiers captures keyboard and mouse modifier state for an event.
type Modifiers struct {
	Ctrl   bool
	Meta   bool
	Shift  bool
	Alt    bool
	Button int
}

// Event represents a DOM event payload delivered to the server.
type Event struct {
	Name    string
	Value   string
	Payload map[string]any
	Form    map[string]string
	Mods    Modifiers
}

// Updates represents state changes returned by event handlers.
// Nil means no updates (but may have side effects).
type Updates any

// Handler represents an event handler with its function and configuration.
type Handler struct {
	metadata.EventOptions // Embedded: Prevent, Stop, Passive, Debounce, Listen, Props, etc.
	Fn                    func(Event) Updates
}
