package html

import "github.com/eleven-am/pondlive/go/internal/dom"

// VisibilityEvent represents page visibility change events.
type VisibilityEvent struct {
	dom.Event
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
	ref dom.RefListener
}

// NewVisibilityHandler creates a new VisibilityHandler.
func NewVisibilityHandler(ref dom.RefListener) *VisibilityHandler {
	return &VisibilityHandler{ref: ref}
}

// OnVisibilityChange registers a handler for the "visibilitychange" event.
func (h *VisibilityHandler) OnVisibilityChange(handler func(VisibilityEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildVisibilityEvent(evt)) }
	h.ref.AddListener("visibilitychange", wrapped, VisibilityEvent{}.props())
}

func buildVisibilityEvent(evt dom.Event) VisibilityEvent {
	detail := extractDetail(evt.Payload)
	return VisibilityEvent{
		Event:           evt,
		Hidden:          payloadBool(detail, "document.hidden", false),
		VisibilityState: PayloadString(detail, "document.visibilityState", ""),
	}
}
