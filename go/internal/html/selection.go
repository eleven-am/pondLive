package html

// SelectionEvent represents text selection events.
type SelectionEvent struct {
	Event
}

// Props returns the list of properties this event needs from the client.
func (SelectionEvent) props() []string {
	return []string{}
}

// SelectionHandler provides selection event handlers.
type SelectionHandler struct {
	ref RefListener
}

// NewSelectionHandler creates a new SelectionHandler.
func NewSelectionHandler(ref RefListener) *SelectionHandler {
	return &SelectionHandler{ref: ref}
}

// OnSelectStart registers a handler for the "selectstart" event.
func (h *SelectionHandler) OnSelectStart(handler func(SelectionEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildSelectionEvent(evt)) }
	h.ref.AddListener("selectstart", wrapped, SelectionEvent{}.props())
}

// OnSelectionChange registers a handler for the "selectionchange" event.
func (h *SelectionHandler) OnSelectionChange(handler func(SelectionEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildSelectionEvent(evt)) }
	h.ref.AddListener("selectionchange", wrapped, SelectionEvent{}.props())
}

func buildSelectionEvent(evt Event) SelectionEvent {
	return SelectionEvent{
		Event: evt,
	}
}
