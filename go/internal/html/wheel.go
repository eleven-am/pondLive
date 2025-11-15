package html

// WheelEvent represents mouse wheel events.
type WheelEvent struct {
	Event
	DeltaX   float64 // Horizontal scroll amount
	DeltaY   float64 // Vertical scroll amount
	DeltaZ   float64 // Z-axis scroll amount
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
}

// Props returns the list of properties this event needs from the client.
func (WheelEvent) props() []string {
	return []string{
		"event.deltaX",
		"event.deltaY",
		"event.deltaZ",
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
	}
}

// WheelHandler provides wheel event handlers.
type WheelHandler struct {
	ref RefListener
}

// NewWheelHandler creates a new WheelHandler.
func NewWheelHandler(ref RefListener) *WheelHandler {
	return &WheelHandler{ref: ref}
}

// OnWheel registers a handler for the "wheel" event.
func (h *WheelHandler) OnWheel(handler func(WheelEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildWheelEvent(evt)) }
	h.ref.AddListener("wheel", wrapped, WheelEvent{}.props())
}

func buildWheelEvent(evt Event) WheelEvent {
	detail := extractDetail(evt.Payload)
	return WheelEvent{
		Event:    evt,
		DeltaX:   payloadFloat(detail, "event.deltaX", 0),
		DeltaY:   payloadFloat(detail, "event.deltaY", 0),
		DeltaZ:   payloadFloat(detail, "event.deltaZ", 0),
		AltKey:   payloadBool(detail, "event.altKey", false),
		CtrlKey:  payloadBool(detail, "event.ctrlKey", false),
		ShiftKey: payloadBool(detail, "event.shiftKey", false),
		MetaKey:  payloadBool(detail, "event.metaKey", false),
	}
}
