package html

// FocusEvent represents focus and blur events.
type FocusEvent struct {
	Event
}

// Props returns the list of properties this event needs from the client.
func (FocusEvent) props() []string {
	return []string{}
}

// FocusHandler provides focus and blur event handlers.
type FocusHandler struct {
	ref RefListener
}

// NewFocusHandler creates a new FocusHandler.
func NewFocusHandler(ref RefListener) *FocusHandler {
	return &FocusHandler{ref: ref}
}

// OnFocus registers a handler for the "focus" event.
func (h *FocusHandler) OnFocus(handler func(FocusEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFocusEvent(evt)) }
	h.ref.AddListener("focus", wrapped, FocusEvent{}.props())
}

// OnBlur registers a handler for the "blur" event.
func (h *FocusHandler) OnBlur(handler func(FocusEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFocusEvent(evt)) }
	h.ref.AddListener("blur", wrapped, FocusEvent{}.props())
}

func buildFocusEvent(evt Event) FocusEvent {
	return FocusEvent{
		Event: evt,
	}
}
