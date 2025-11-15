package html

// CompositionEvent represents IME composition events.
type CompositionEvent struct {
	Event
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
	ref RefListener
}

// NewCompositionHandler creates a new CompositionHandler.
func NewCompositionHandler(ref RefListener) *CompositionHandler {
	return &CompositionHandler{ref: ref}
}

// OnCompositionStart registers a handler for the "compositionstart" event.
func (h *CompositionHandler) OnCompositionStart(handler func(CompositionEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildCompositionEvent(evt)) }
	h.ref.AddListener("compositionstart", wrapped, CompositionEvent{}.props())
}

// OnCompositionUpdate registers a handler for the "compositionupdate" event.
func (h *CompositionHandler) OnCompositionUpdate(handler func(CompositionEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildCompositionEvent(evt)) }
	h.ref.AddListener("compositionupdate", wrapped, CompositionEvent{}.props())
}

// OnCompositionEnd registers a handler for the "compositionend" event.
func (h *CompositionHandler) OnCompositionEnd(handler func(CompositionEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildCompositionEvent(evt)) }
	h.ref.AddListener("compositionend", wrapped, CompositionEvent{}.props())
}

func buildCompositionEvent(evt Event) CompositionEvent {
	detail := extractDetail(evt.Payload)
	return CompositionEvent{
		Event: evt,
		Data:  PayloadString(detail, "event.data", ""),
	}
}
