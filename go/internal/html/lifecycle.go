package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// LifecycleEvent represents page lifecycle events (pageshow, pagehide, beforeunload).
type LifecycleEvent struct {
	dom2.Event
	Persisted bool // For pageshow/pagehide: whether page is being persisted in bfcache
}

// Props returns the list of properties this event needs from the client.
func (LifecycleEvent) props() []string {
	return []string{
		"event.persisted",
	}
}

// LifecycleHandler provides page lifecycle event handlers.
type LifecycleHandler struct {
	ref dom2.RefListener
}

// NewLifecycleHandler creates a new LifecycleHandler.
func NewLifecycleHandler(ref dom2.RefListener) *LifecycleHandler {
	return &LifecycleHandler{ref: ref}
}

// OnPageShow registers a handler for the "pageshow" event.
func (h *LifecycleHandler) OnPageShow(handler func(LifecycleEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildLifecycleEvent(evt)) }
	h.ref.AddListener("pageshow", wrapped, LifecycleEvent{}.props())
}

// OnPageHide registers a handler for the "pagehide" event.
func (h *LifecycleHandler) OnPageHide(handler func(LifecycleEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildLifecycleEvent(evt)) }
	h.ref.AddListener("pagehide", wrapped, LifecycleEvent{}.props())
}

// OnBeforeUnload registers a handler for the "beforeunload" event.
func (h *LifecycleHandler) OnBeforeUnload(handler func(LifecycleEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildLifecycleEvent(evt)) }
	h.ref.AddListener("beforeunload", wrapped, LifecycleEvent{}.props())
}

func buildLifecycleEvent(evt dom2.Event) LifecycleEvent {
	detail := extractDetail(evt.Payload)
	return LifecycleEvent{
		Event:     evt,
		Persisted: payloadBool(detail, "event.persisted", false),
	}
}
