package html

// DialogEvent represents dialog events (close, cancel).
type DialogEvent struct {
	Event
}

// Props returns the list of properties this event needs from the client.
func (DialogEvent) props() []string {
	return []string{}
}

// DialogHandler provides dialog event handlers.
type DialogHandler struct {
	ref RefListener
}

// NewDialogHandler creates a new DialogHandler.
func NewDialogHandler(ref RefListener) *DialogHandler {
	return &DialogHandler{ref: ref}
}

// OnClose registers a handler for the "close" event.
func (h *DialogHandler) OnClose(handler func(DialogEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDialogEvent(evt)) }
	h.ref.AddListener("close", wrapped, DialogEvent{}.props())
}

// OnCancel registers a handler for the "cancel" event.
func (h *DialogHandler) OnCancel(handler func(DialogEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDialogEvent(evt)) }
	h.ref.AddListener("cancel", wrapped, DialogEvent{}.props())
}

func buildDialogEvent(evt Event) DialogEvent {
	return DialogEvent{
		Event: evt,
	}
}
