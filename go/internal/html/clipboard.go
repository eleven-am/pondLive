package html

// ClipboardEvent represents clipboard events (copy, cut, paste).
type ClipboardEvent struct {
	Event
	// Note: ClipboardData is complex and requires special handling
	// For now, just basic event structure
}

// Props returns the list of properties this event needs from the client.
func (ClipboardEvent) props() []string {
	return []string{}
}

// ClipboardHandler provides clipboard event handlers.
type ClipboardHandler struct {
	ref RefListener
}

// NewClipboardHandler creates a new ClipboardHandler.
func NewClipboardHandler(ref RefListener) *ClipboardHandler {
	return &ClipboardHandler{ref: ref}
}

// OnCopy registers a handler for the "copy" event.
func (h *ClipboardHandler) OnCopy(handler func(ClipboardEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClipboardEvent(evt)) }
	h.ref.AddListener("copy", wrapped, ClipboardEvent{}.props())
}

// OnCut registers a handler for the "cut" event.
func (h *ClipboardHandler) OnCut(handler func(ClipboardEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClipboardEvent(evt)) }
	h.ref.AddListener("cut", wrapped, ClipboardEvent{}.props())
}

// OnPaste registers a handler for the "paste" event.
func (h *ClipboardHandler) OnPaste(handler func(ClipboardEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClipboardEvent(evt)) }
	h.ref.AddListener("paste", wrapped, ClipboardEvent{}.props())
}

func buildClipboardEvent(evt Event) ClipboardEvent {
	return ClipboardEvent{
		Event: evt,
	}
}
