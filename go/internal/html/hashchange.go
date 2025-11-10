package html

// HashChangeEvent represents URL hash change events.
type HashChangeEvent struct {
	Event
	OldURL string // Previous URL
	NewURL string // New URL
}

// Props returns the list of properties this event needs from the client.
func (HashChangeEvent) props() []string {
	return []string{
		"event.oldURL",
		"event.newURL",
	}
}

// HashChangeHandler provides hash change event handlers.
type HashChangeHandler struct {
	ref RefListener
}

// NewHashChangeHandler creates a new HashChangeHandler.
func NewHashChangeHandler(ref RefListener) *HashChangeHandler {
	return &HashChangeHandler{ref: ref}
}

// OnHashChange registers a handler for the "hashchange" event.
func (h *HashChangeHandler) OnHashChange(handler func(HashChangeEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildHashChangeEvent(evt)) }
	h.ref.AddListener("hashchange", wrapped, HashChangeEvent{}.props())
}

func buildHashChangeEvent(evt Event) HashChangeEvent {
	return HashChangeEvent{
		Event:  evt,
		OldURL: payloadString(evt.Payload, "event.oldURL", ""),
		NewURL: payloadString(evt.Payload, "event.newURL", ""),
	}
}
