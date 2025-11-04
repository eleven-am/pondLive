package runtime

import handlers "github.com/eleven-am/pondlive/go/internal/handlers"

// WireEvent represents the payload received from the client for an event.
type WireEvent struct {
	Name    string            `json:"name"`
	Value   string            `json:"value,omitempty"`
	Payload map[string]any    `json:"payload,omitempty"`
	Form    map[string]string `json:"form,omitempty"`
	Mods    WireModifiers     `json:"mods,omitempty"`
}

// WireModifiers captures modifier state from the client payload.
type WireModifiers struct {
	Ctrl   bool `json:"ctrl,omitempty"`
	Meta   bool `json:"meta,omitempty"`
	Shift  bool `json:"shift,omitempty"`
	Alt    bool `json:"alt,omitempty"`
	Button int  `json:"button,omitempty"`
}

// ToEvent converts a wire payload into the handler event type.
func (w WireEvent) ToEvent() handlers.Event {
	return handlers.Event{
		Name:    w.Name,
		Value:   w.Value,
		Payload: cloneAnyMap(w.Payload),
		Form:    cloneStringMap(w.Form),
		Mods: handlers.Modifiers{
			Ctrl:   w.Mods.Ctrl,
			Meta:   w.Mods.Meta,
			Shift:  w.Mods.Shift,
			Alt:    w.Mods.Alt,
			Button: w.Mods.Button,
		},
	}
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
