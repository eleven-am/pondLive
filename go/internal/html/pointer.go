package html

// PointerEvent represents pointer events (pointerdown, pointerup, pointermove, etc).
type PointerEvent struct {
	Event
	PointerType string  // Type of pointer (mouse, pen, touch)
	PointerID   int     // Unique pointer identifier
	Button      int     // Which button
	Buttons     int     // Which buttons are pressed
	ClientX     float64 // X coordinate relative to viewport
	ClientY     float64 // Y coordinate relative to viewport
	MovementX   float64 // X movement since last event
	MovementY   float64 // Y movement since last event
	OffsetX     float64 // X coordinate relative to target
	OffsetY     float64 // Y coordinate relative to target
	PageX       float64 // X coordinate relative to page
	PageY       float64 // Y coordinate relative to page
	ScreenX     float64 // X coordinate relative to screen
	ScreenY     float64 // Y coordinate relative to screen
	IsPrimary   bool    // Is primary pointer
	AltKey      bool    // Alt key pressed
	CtrlKey     bool    // Control key pressed
	ShiftKey    bool    // Shift key pressed
	MetaKey     bool    // Meta key pressed
}

// Props returns the list of properties this event needs from the client.
func (PointerEvent) props() []string {
	return []string{
		"event.pointerType",
		"event.pointerId",
		"event.button",
		"event.buttons",
		"event.clientX",
		"event.clientY",
		"event.movementX",
		"event.movementY",
		"event.offsetX",
		"event.offsetY",
		"event.pageX",
		"event.pageY",
		"event.screenX",
		"event.screenY",
		"event.isPrimary",
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
	}
}

// PointerHandler provides pointer event handlers.
type PointerHandler struct {
	ref RefListener
}

// NewPointerHandler creates a new PointerHandler.
func NewPointerHandler(ref RefListener) *PointerHandler {
	return &PointerHandler{ref: ref}
}

// OnPointerDown registers a handler for the "pointerdown" event.
func (h *PointerHandler) OnPointerDown(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointerdown", wrapped, PointerEvent{}.props())
}

// OnPointerUp registers a handler for the "pointerup" event.
func (h *PointerHandler) OnPointerUp(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointerup", wrapped, PointerEvent{}.props())
}

// OnPointerMove registers a handler for the "pointermove" event.
func (h *PointerHandler) OnPointerMove(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointermove", wrapped, PointerEvent{}.props())
}

// OnPointerEnter registers a handler for the "pointerenter" event.
func (h *PointerHandler) OnPointerEnter(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointerenter", wrapped, PointerEvent{}.props())
}

// OnPointerLeave registers a handler for the "pointerleave" event.
func (h *PointerHandler) OnPointerLeave(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointerleave", wrapped, PointerEvent{}.props())
}

// OnPointerOver registers a handler for the "pointerover" event.
func (h *PointerHandler) OnPointerOver(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointerover", wrapped, PointerEvent{}.props())
}

// OnPointerOut registers a handler for the "pointerout" event.
func (h *PointerHandler) OnPointerOut(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointerout", wrapped, PointerEvent{}.props())
}

// OnPointerCancel registers a handler for the "pointercancel" event.
func (h *PointerHandler) OnPointerCancel(handler func(PointerEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildPointerEvent(evt)) }
	h.ref.AddListener("pointercancel", wrapped, PointerEvent{}.props())
}

func buildPointerEvent(evt Event) PointerEvent {
	return PointerEvent{
		Event:       evt,
		PointerType: payloadString(evt.Payload, "event.pointerType", ""),
		PointerID:   payloadInt(evt.Payload, "event.pointerId", 0),
		Button:      payloadInt(evt.Payload, "event.button", 0),
		Buttons:     payloadInt(evt.Payload, "event.buttons", 0),
		ClientX:     payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:     payloadFloat(evt.Payload, "event.clientY", 0),
		MovementX:   payloadFloat(evt.Payload, "event.movementX", 0),
		MovementY:   payloadFloat(evt.Payload, "event.movementY", 0),
		OffsetX:     payloadFloat(evt.Payload, "event.offsetX", 0),
		OffsetY:     payloadFloat(evt.Payload, "event.offsetY", 0),
		PageX:       payloadFloat(evt.Payload, "event.pageX", 0),
		PageY:       payloadFloat(evt.Payload, "event.pageY", 0),
		ScreenX:     payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:     payloadFloat(evt.Payload, "event.screenY", 0),
		IsPrimary:   payloadBool(evt.Payload, "event.isPrimary", false),
		AltKey:      payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:     payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey:    payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:     payloadBool(evt.Payload, "event.metaKey", false),
	}
}
