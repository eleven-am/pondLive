package html

// ResizeEvent represents window/element resize events.
type ResizeEvent struct {
	Event
	// Note: ResizeObserver contentRect would need special handling
	// For now, just basic event structure
}

// Props returns the list of properties this event needs from the client.
func (ResizeEvent) props() []string {
	return []string{}
}

// ResizeHandler provides resize event handlers.
type ResizeHandler struct {
	ref RefListener
}

// NewResizeHandler creates a new ResizeHandler.
func NewResizeHandler(ref RefListener) *ResizeHandler {
	return &ResizeHandler{ref: ref}
}

// OnResize registers a handler for the "resize" event.
func (h *ResizeHandler) OnResize(handler func(ResizeEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildResizeEvent(evt)) }
	h.ref.AddListener("resize", wrapped, ResizeEvent{}.props())
}

func buildResizeEvent(evt Event) ResizeEvent {
	return ResizeEvent{
		Event: evt,
	}
}
