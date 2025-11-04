package html

// Event represents a DOM event payload delivered to the server.
type Event struct {
	Name    string
	Value   string
	Payload map[string]any
	Form    map[string]string
	Mods    Modifiers
}

// Modifiers captures keyboard and mouse modifier state for an event.
type Modifiers struct {
	Ctrl   bool
	Meta   bool
	Shift  bool
	Alt    bool
	Button int
}

// Updates marks a handler return value that can trigger rerenders.
type Updates interface{ isUpdates() }

// EventHandler represents a server-side event handler for a DOM event.
type EventHandler func(Event) Updates

type rerender struct{}

func (rerender) isUpdates() {}

// Rerender signals that the component tree should be rendered again.
func Rerender() Updates { return rerender{} }
