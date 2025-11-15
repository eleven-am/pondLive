package html

// ToggleEvent represents toggle events (for details/summary elements).
type ToggleEvent struct {
	Event
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
	ref RefListener
}

// NewToggleHandler creates a new ToggleHandler.
func NewToggleHandler(ref RefListener) *ToggleHandler {
	return &ToggleHandler{ref: ref}
}

// OnToggle registers a handler for the "toggle" event.
func (h *ToggleHandler) OnToggle(handler func(ToggleEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildToggleEvent(evt)) }
	h.ref.AddListener("toggle", wrapped, ToggleEvent{}.props())
}

func buildToggleEvent(evt Event) ToggleEvent {
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
