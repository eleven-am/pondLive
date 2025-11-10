package html

// VisibilityEvent represents page visibility change events.
type VisibilityEvent struct {
	Event
	Hidden          bool   // Is page hidden
	VisibilityState string // Visibility state (visible, hidden, prerender)
}

// Props returns the list of properties this event needs from the client.
func (VisibilityEvent) props() []string {
	return []string{
		"document.hidden",
		"document.visibilityState",
	}
}

// VisibilityHandler provides visibility event handlers.
type VisibilityHandler struct {
	ref RefListener
}

// NewVisibilityHandler creates a new VisibilityHandler.
func NewVisibilityHandler(ref RefListener) *VisibilityHandler {
	return &VisibilityHandler{ref: ref}
}

// OnVisibilityChange registers a handler for the "visibilitychange" event.
func (h *VisibilityHandler) OnVisibilityChange(handler func(VisibilityEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildVisibilityEvent(evt)) }
	h.ref.AddListener("visibilitychange", wrapped, VisibilityEvent{}.props())
}

func buildVisibilityEvent(evt Event) VisibilityEvent {
	return VisibilityEvent{
		Event:           evt,
		Hidden:          payloadBool(evt.Payload, "document.hidden", false),
		VisibilityState: PayloadString(evt.Payload, "document.visibilityState", ""),
	}
}
