package html

// DragEvent represents drag and drop events.
type DragEvent struct {
	Event
	ClientX  float64 // X coordinate relative to viewport
	ClientY  float64 // Y coordinate relative to viewport
	ScreenX  float64 // X coordinate relative to screen
	ScreenY  float64 // Y coordinate relative to screen
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
}

// Props returns the list of properties this event needs from the client.
func (DragEvent) props() []string {
	return []string{
		"event.clientX",
		"event.clientY",
		"event.screenX",
		"event.screenY",
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
	}
}

// DragHandler provides drag and drop event handlers.
type DragHandler struct {
	ref RefListener
}

// NewDragHandler creates a new DragHandler.
func NewDragHandler(ref RefListener) *DragHandler {
	return &DragHandler{ref: ref}
}

// OnDrag registers a handler for the "drag" event.
func (h *DragHandler) OnDrag(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("drag", wrapped, DragEvent{}.props())
}

// OnDragStart registers a handler for the "dragstart" event.
func (h *DragHandler) OnDragStart(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("dragstart", wrapped, DragEvent{}.props())
}

// OnDragEnd registers a handler for the "dragend" event.
func (h *DragHandler) OnDragEnd(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("dragend", wrapped, DragEvent{}.props())
}

// OnDragEnter registers a handler for the "dragenter" event.
func (h *DragHandler) OnDragEnter(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("dragenter", wrapped, DragEvent{}.props())
}

// OnDragLeave registers a handler for the "dragleave" event.
func (h *DragHandler) OnDragLeave(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("dragleave", wrapped, DragEvent{}.props())
}

// OnDragOver registers a handler for the "dragover" event.
func (h *DragHandler) OnDragOver(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("dragover", wrapped, DragEvent{}.props())
}

// OnDrop registers a handler for the "drop" event.
func (h *DragHandler) OnDrop(handler func(DragEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDragEvent(evt)) }
	h.ref.AddListener("drop", wrapped, DragEvent{}.props())
}

func buildDragEvent(evt Event) DragEvent {
	return DragEvent{
		Event:    evt,
		ClientX:  payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:  payloadFloat(evt.Payload, "event.clientY", 0),
		ScreenX:  payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:  payloadFloat(evt.Payload, "event.screenY", 0),
		AltKey:   payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:  payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey: payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:  payloadBool(evt.Payload, "event.metaKey", false),
	}
}
