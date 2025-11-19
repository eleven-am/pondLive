package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// FullscreenEvent represents fullscreen change events.
type FullscreenEvent struct {
	dom2.Event
}

// Props returns the list of properties this event needs from the client.
func (FullscreenEvent) props() []string {
	return []string{}
}

// FullscreenHandler provides fullscreen event handlers.
type FullscreenHandler struct {
	ref dom2.RefListener
}

// NewFullscreenHandler creates a new FullscreenHandler.
func NewFullscreenHandler(ref dom2.RefListener) *FullscreenHandler {
	return &FullscreenHandler{ref: ref}
}

// OnFullscreenChange registers a handler for the "fullscreenchange" event.
func (h *FullscreenHandler) OnFullscreenChange(handler func(FullscreenEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildFullscreenEvent(evt)) }
	h.ref.AddListener("fullscreenchange", wrapped, FullscreenEvent{}.props())
}

// OnFullscreenError registers a handler for the "fullscreenerror" event.
func (h *FullscreenHandler) OnFullscreenError(handler func(FullscreenEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildFullscreenEvent(evt)) }
	h.ref.AddListener("fullscreenerror", wrapped, FullscreenEvent{}.props())
}

func buildFullscreenEvent(evt dom2.Event) FullscreenEvent {
	return FullscreenEvent{
		Event: evt,
	}
}
