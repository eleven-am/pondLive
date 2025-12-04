package metadata

type HandlerMeta struct {
	Event   string `json:"event"`
	Handler string `json:"handler"`
	EventOptions
}

type EventOptions struct {
	Prevent  bool     `json:"prevent,omitempty"`
	Stop     bool     `json:"stop,omitempty"`
	Passive  bool     `json:"passive,omitempty"`
	Once     bool     `json:"once,omitempty"`
	Capture  bool     `json:"capture,omitempty"`
	Debounce int      `json:"debounce,omitempty"`
	Throttle int      `json:"throttle,omitempty"`
	Listen   []string `json:"listen,omitempty"`
	Props    []string `json:"props,omitempty"`
}
