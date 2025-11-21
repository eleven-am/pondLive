package html

import "github.com/eleven-am/pondlive/go/internal/dom"

// ToggleEvent represents toggle events (for details/summary elements).
type ToggleEvent struct {
	dom.Event
	NewState string // "open" or "closed"
}

// Props returns the list of properties this event needs from the client.
func (ToggleEvent) props() []string {
	return []string{
		"target.open",
	}
}

// ToggleHandler provides toggle event handlers.
type ToggleHandler struct {
	ref dom.RefListener
}

// NewToggleHandler creates a new ToggleHandler.
func NewToggleHandler(ref dom.RefListener) *ToggleHandler {
	return &ToggleHandler{ref: ref}
}

// OnToggle registers a handler for the "toggle" event.
func (h *ToggleHandler) OnToggle(handler func(ToggleEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildToggleEvent(evt)) }
	h.ref.AddListener("toggle", wrapped, ToggleEvent{}.props())
}

func buildToggleEvent(evt dom.Event) ToggleEvent {
	detail := extractDetail(evt.Payload)
	open := payloadBool(detail, "target.open", false)
	newState := "closed"
	if open {
		newState = "open"
	}
	return ToggleEvent{
		Event:    evt,
		NewState: newState,
	}
}
