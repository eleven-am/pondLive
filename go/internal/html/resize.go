package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// ResizeEvent represents window/element resize events.
type ResizeEvent struct {
	dom2.Event
	// Note: ResizeObserver contentRect would need special handling
	// For now, just basic event structure
}

// Props returns the list of properties this event needs from the client.
func (ResizeEvent) props() []string {
	return []string{}
}

// ResizeHandler provides resize event handlers.
type ResizeHandler struct {
	ref dom2.RefListener
}

// NewResizeHandler creates a new ResizeHandler.
func NewResizeHandler(ref dom2.RefListener) *ResizeHandler {
	return &ResizeHandler{ref: ref}
}

// OnResize registers a handler for the "resize" event.
func (h *ResizeHandler) OnResize(handler func(ResizeEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildResizeEvent(evt)) }
	h.ref.AddListener("resize", wrapped, ResizeEvent{}.props())
}

func buildResizeEvent(evt dom2.Event) ResizeEvent {
	return ResizeEvent{
		Event: evt,
	}
}
