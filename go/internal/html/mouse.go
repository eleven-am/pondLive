package html

// MouseEvent represents mouse events (mousedown, mouseup, mousemove, etc).
type MouseEvent struct {
	Event
	Button    int     // Which mouse button
	Buttons   int     // Which buttons are pressed
	ClientX   float64 // X coordinate relative to viewport
	ClientY   float64 // Y coordinate relative to viewport
	ScreenX   float64 // X coordinate relative to screen
	ScreenY   float64 // Y coordinate relative to screen
	MovementX float64 // X movement since last event
	MovementY float64 // Y movement since last event
	OffsetX   float64 // X coordinate relative to target
	OffsetY   float64 // Y coordinate relative to target
	PageX     float64 // X coordinate relative to page
	PageY     float64 // Y coordinate relative to page
	AltKey    bool    // Alt key pressed
	CtrlKey   bool    // Control key pressed
	ShiftKey  bool    // Shift key pressed
	MetaKey   bool    // Meta key pressed
}

// Props returns the list of properties this event needs from the client.
func (MouseEvent) props() []string {
	return []string{
		"event.button",
		"event.buttons",
		"event.clientX",
		"event.clientY",
		"event.screenX",
		"event.screenY",
		"event.movementX",
		"event.movementY",
		"event.offsetX",
		"event.offsetY",
		"event.pageX",
		"event.pageY",
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
	}
}

// MouseHandler provides mouse event handlers.
type MouseHandler struct {
	ref RefListener
}

// NewMouseHandler creates a new MouseHandler.
func NewMouseHandler(ref RefListener) *MouseHandler {
	return &MouseHandler{ref: ref}
}

// OnMouseDown registers a handler for the "mousedown" event.
func (h *MouseHandler) OnMouseDown(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mousedown", wrapped, MouseEvent{}.props())
}

// OnMouseUp registers a handler for the "mouseup" event.
func (h *MouseHandler) OnMouseUp(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mouseup", wrapped, MouseEvent{}.props())
}

// OnMouseMove registers a handler for the "mousemove" event.
func (h *MouseHandler) OnMouseMove(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mousemove", wrapped, MouseEvent{}.props())
}

// OnMouseEnter registers a handler for the "mouseenter" event.
func (h *MouseHandler) OnMouseEnter(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mouseenter", wrapped, MouseEvent{}.props())
}

// OnMouseLeave registers a handler for the "mouseleave" event.
func (h *MouseHandler) OnMouseLeave(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mouseleave", wrapped, MouseEvent{}.props())
}

// OnMouseOver registers a handler for the "mouseover" event.
func (h *MouseHandler) OnMouseOver(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mouseover", wrapped, MouseEvent{}.props())
}

// OnMouseOut registers a handler for the "mouseout" event.
func (h *MouseHandler) OnMouseOut(handler func(MouseEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMouseEvent(evt)) }
	h.ref.AddListener("mouseout", wrapped, MouseEvent{}.props())
}

func buildMouseEvent(evt Event) MouseEvent {
	return MouseEvent{
		Event:     evt,
		Button:    payloadInt(evt.Payload, "event.button", 0),
		Buttons:   payloadInt(evt.Payload, "event.buttons", 0),
		ClientX:   payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:   payloadFloat(evt.Payload, "event.clientY", 0),
		ScreenX:   payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:   payloadFloat(evt.Payload, "event.screenY", 0),
		MovementX: payloadFloat(evt.Payload, "event.movementX", 0),
		MovementY: payloadFloat(evt.Payload, "event.movementY", 0),
		OffsetX:   payloadFloat(evt.Payload, "event.offsetX", 0),
		OffsetY:   payloadFloat(evt.Payload, "event.offsetY", 0),
		PageX:     payloadFloat(evt.Payload, "event.pageX", 0),
		PageY:     payloadFloat(evt.Payload, "event.pageY", 0),
		AltKey:    payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:   payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey:  payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:   payloadBool(evt.Payload, "event.metaKey", false),
	}
}
