package html

// TouchEvent represents touch events (touchstart, touchend, touchmove, touchcancel).
type TouchEvent struct {
	Event
	AltKey   bool // Alt key pressed
	CtrlKey  bool // Control key pressed
	ShiftKey bool // Shift key pressed
	MetaKey  bool // Meta key pressed
	// Note: Touch lists (touches, targetTouches, changedTouches) would need
	// special handling as they are arrays. For now, we just capture modifier keys.
}

// Props returns the list of properties this event needs from the client.
func (TouchEvent) props() []string {
	return []string{
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
	}
}

// TouchHandler provides touch event handlers.
type TouchHandler struct {
	ref RefListener
}

// NewTouchHandler creates a new TouchHandler.
func NewTouchHandler(ref RefListener) *TouchHandler {
	return &TouchHandler{ref: ref}
}

// OnTouchStart registers a handler for the "touchstart" event.
func (h *TouchHandler) OnTouchStart(handler func(TouchEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	h.ref.AddListener("touchstart", wrapped, TouchEvent{}.props())
}

// OnTouchEnd registers a handler for the "touchend" event.
func (h *TouchHandler) OnTouchEnd(handler func(TouchEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	h.ref.AddListener("touchend", wrapped, TouchEvent{}.props())
}

// OnTouchMove registers a handler for the "touchmove" event.
func (h *TouchHandler) OnTouchMove(handler func(TouchEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	h.ref.AddListener("touchmove", wrapped, TouchEvent{}.props())
}

// OnTouchCancel registers a handler for the "touchcancel" event.
func (h *TouchHandler) OnTouchCancel(handler func(TouchEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildTouchEvent(evt)) }
	h.ref.AddListener("touchcancel", wrapped, TouchEvent{}.props())
}

func buildTouchEvent(evt Event) TouchEvent {
	return TouchEvent{
		Event:    evt,
		AltKey:   payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:  payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey: payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:  payloadBool(evt.Payload, "event.metaKey", false),
	}
}
