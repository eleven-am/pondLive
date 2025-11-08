package html

// ScrollEvent represents scroll events.
type ScrollEvent struct {
	Event
	ScrollTop    float64 // Current vertical scroll position
	ScrollLeft   float64 // Current horizontal scroll position
	ScrollHeight float64 // Total scrollable height
	ScrollWidth  float64 // Total scrollable width
	ClientHeight float64 // Visible height
	ClientWidth  float64 // Visible width
}

// Props returns the list of properties this event needs from the client.
func (ScrollEvent) props() []string {
	return []string{
		"target.scrollTop",
		"target.scrollLeft",
		"target.scrollHeight",
		"target.scrollWidth",
		"target.clientHeight",
		"target.clientWidth",
	}
}

// ScrollHandler provides scroll event handlers.
type ScrollHandler struct {
	ref RefListener
}

// NewScrollHandler creates a new ScrollHandler.
func NewScrollHandler(ref RefListener) *ScrollHandler {
	return &ScrollHandler{ref: ref}
}

// OnScroll registers a handler for the "scroll" event.
func (h *ScrollHandler) OnScroll(handler func(ScrollEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildScrollEvent(evt)) }
	h.ref.AddListener("scroll", wrapped, ScrollEvent{}.props())
}

func buildScrollEvent(evt Event) ScrollEvent {
	return ScrollEvent{
		Event:        evt,
		ScrollTop:    payloadFloat(evt.Payload, "target.scrollTop", 0),
		ScrollLeft:   payloadFloat(evt.Payload, "target.scrollLeft", 0),
		ScrollHeight: payloadFloat(evt.Payload, "target.scrollHeight", 0),
		ScrollWidth:  payloadFloat(evt.Payload, "target.scrollWidth", 0),
		ClientHeight: payloadFloat(evt.Payload, "target.clientHeight", 0),
		ClientWidth:  payloadFloat(evt.Payload, "target.clientWidth", 0),
	}
}
