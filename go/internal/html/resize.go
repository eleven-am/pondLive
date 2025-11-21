package html

import "github.com/eleven-am/pondlive/go/internal/dom"

// ResizeEvent represents window/element resize events.
type ResizeEvent struct {
	dom.Event
	// Note: ResizeObserver contentRect would need special handling
	// For now, just basic event structure
}

// Props returns the list of properties this event needs from the client.
func (ResizeEvent) props() []string {
	return []string{}
}

// ResizeHandler provides resize event handlers.
type ResizeHandler struct {
	ref dom.RefListener
}

// NewResizeHandler creates a new ResizeHandler.
func NewResizeHandler(ref dom.RefListener) *ResizeHandler {
	return &ResizeHandler{ref: ref}
}

// OnResize registers a handler for the "resize" event.
func (h *ResizeHandler) OnResize(handler func(ResizeEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildResizeEvent(evt)) }
	h.ref.AddListener("resize", wrapped, ResizeEvent{}.props())
}

func buildResizeEvent(evt dom.Event) ResizeEvent {
	return ResizeEvent{
		Event: evt,
	}
}
