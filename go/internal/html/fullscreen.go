package html

// FullscreenEvent represents fullscreen change events.
type FullscreenEvent struct {
	Event
}

// Props returns the list of properties this event needs from the client.
func (FullscreenEvent) props() []string {
	return []string{}
}

// FullscreenHandler provides fullscreen event handlers.
type FullscreenHandler struct {
	ref RefListener
}

// NewFullscreenHandler creates a new FullscreenHandler.
func NewFullscreenHandler(ref RefListener) *FullscreenHandler {
	return &FullscreenHandler{ref: ref}
}

// OnFullscreenChange registers a handler for the "fullscreenchange" event.
func (h *FullscreenHandler) OnFullscreenChange(handler func(FullscreenEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFullscreenEvent(evt)) }
	h.ref.AddListener("fullscreenchange", wrapped, FullscreenEvent{}.props())
}

// OnFullscreenError registers a handler for the "fullscreenerror" event.
func (h *FullscreenHandler) OnFullscreenError(handler func(FullscreenEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFullscreenEvent(evt)) }
	h.ref.AddListener("fullscreenerror", wrapped, FullscreenEvent{}.props())
}

func buildFullscreenEvent(evt Event) FullscreenEvent {
	return FullscreenEvent{
		Event: evt,
	}
}
