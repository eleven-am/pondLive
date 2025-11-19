package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// CompositionEvent represents IME composition events.
type CompositionEvent struct {
	dom2.Event
	Data string // Composition data
}

// Props returns the list of properties this event needs from the client.
func (CompositionEvent) props() []string {
	return []string{
		"event.data",
	}
}

// CompositionHandler provides composition event handlers.
type CompositionHandler struct {
	ref dom2.RefListener
}

// NewCompositionHandler creates a new CompositionHandler.
func NewCompositionHandler(ref dom2.RefListener) *CompositionHandler {
	return &CompositionHandler{ref: ref}
}

// OnCompositionStart registers a handler for the "compositionstart" event.
func (h *CompositionHandler) OnCompositionStart(handler func(CompositionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildCompositionEvent(evt)) }
	h.ref.AddListener("compositionstart", wrapped, CompositionEvent{}.props())
}

// OnCompositionUpdate registers a handler for the "compositionupdate" event.
func (h *CompositionHandler) OnCompositionUpdate(handler func(CompositionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildCompositionEvent(evt)) }
	h.ref.AddListener("compositionupdate", wrapped, CompositionEvent{}.props())
}

// OnCompositionEnd registers a handler for the "compositionend" event.
func (h *CompositionHandler) OnCompositionEnd(handler func(CompositionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildCompositionEvent(evt)) }
	h.ref.AddListener("compositionend", wrapped, CompositionEvent{}.props())
}

func buildCompositionEvent(evt dom2.Event) CompositionEvent {
	detail := extractDetail(evt.Payload)
	return CompositionEvent{
		Event: evt,
		Data:  PayloadString(detail, "event.data", ""),
	}
}
