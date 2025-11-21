package html

import "github.com/eleven-am/pondlive/go/internal/dom"

// HashChangeEvent represents URL hash change events.
type HashChangeEvent struct {
	dom.Event
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
	ref dom.RefListener
}

// NewHashChangeHandler creates a new HashChangeHandler.
func NewHashChangeHandler(ref dom.RefListener) *HashChangeHandler {
	return &HashChangeHandler{ref: ref}
}

// OnHashChange registers a handler for the "hashchange" event.
func (h *HashChangeHandler) OnHashChange(handler func(HashChangeEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildHashChangeEvent(evt)) }
	h.ref.AddListener("hashchange", wrapped, HashChangeEvent{}.props())
}

func buildHashChangeEvent(evt dom.Event) HashChangeEvent {
	detail := extractDetail(evt.Payload)
	return HashChangeEvent{
		Event:  evt,
		OldURL: PayloadString(detail, "event.oldURL", ""),
		NewURL: PayloadString(detail, "event.newURL", ""),
	}
}
