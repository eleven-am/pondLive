package metadata

// HandlerMeta describes an event handler attachment.
type HandlerMeta struct {
	Event   string `json:"event"`   // "click", "input", etc
	Handler string `json:"handler"` // Handler ID (assigned by runtime)
	EventOptions
}

// EventOptions configures event handler behavior.
type EventOptions struct {
	Prevent  bool     `json:"prevent,omitempty"`
	Stop     bool     `json:"stop,omitempty"`
	Passive  bool     `json:"passive,omitempty"`
	Once     bool     `json:"once,omitempty"`
	Capture  bool     `json:"capture,omitempty"`
	Debounce int      `json:"debounce,omitempty"` // ms
	Throttle int      `json:"throttle,omitempty"` // ms
	Listen   []string `json:"listen,omitempty"`   // Additional events to listen to
	Props    []string `json:"props,omitempty"`    // Event properties to capture from DOM
}
