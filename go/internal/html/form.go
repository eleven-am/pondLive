package html

// FormEvent represents form events (submit, reset, invalid).
type FormEvent struct {
	Event
}

// Props returns the list of properties this event needs from the client.
func (FormEvent) props() []string {
	return []string{}
}

// FormHandler provides form event handlers.
type FormHandler struct {
	ref RefListener
}

// NewFormHandler creates a new FormHandler.
func NewFormHandler(ref RefListener) *FormHandler {
	return &FormHandler{ref: ref}
}

// OnSubmit registers a handler for the "submit" event.
func (h *FormHandler) OnSubmit(handler func(FormEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFormEvent(evt)) }
	h.ref.AddListener("submit", wrapped, FormEvent{}.props())
}

// OnReset registers a handler for the "reset" event.
func (h *FormHandler) OnReset(handler func(FormEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFormEvent(evt)) }
	h.ref.AddListener("reset", wrapped, FormEvent{}.props())
}

// OnInvalid registers a handler for the "invalid" event.
func (h *FormHandler) OnInvalid(handler func(FormEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFormEvent(evt)) }
	h.ref.AddListener("invalid", wrapped, FormEvent{}.props())
}

func buildFormEvent(evt Event) FormEvent {
	return FormEvent{
		Event: evt,
	}
}
