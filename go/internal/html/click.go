package html

// ClickEvent represents mouse click events (click, dblclick, contextmenu).
type ClickEvent struct {
	Event
	Detail   int     // Number of clicks
	AltKey   bool    // Alt key pressed
	CtrlKey  bool    // Control key pressed
	ShiftKey bool    // Shift key pressed
	MetaKey  bool    // Meta key pressed
	Button   int     // Which mouse button
	Buttons  int     // Which buttons are pressed
	ClientX  float64 // X coordinate relative to viewport
	ClientY  float64 // Y coordinate relative to viewport
	OffsetX  float64 // X coordinate relative to target
	OffsetY  float64 // Y coordinate relative to target
	PageX    float64 // X coordinate relative to page
	PageY    float64 // Y coordinate relative to page
	ScreenX  float64 // X coordinate relative to screen
	ScreenY  float64 // Y coordinate relative to screen
}

// Props returns the list of properties this event needs from the client.
func (ClickEvent) props() []string {
	return []string{
		"event.detail",
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
		"event.button",
		"event.buttons",
		"event.clientX",
		"event.clientY",
		"event.offsetX",
		"event.offsetY",
		"event.pageX",
		"event.pageY",
		"event.screenX",
		"event.screenY",
	}
}

// ClickHandler provides click-related event handlers.
type ClickHandler struct {
	ref RefListener
}

// NewClickHandler creates a new ClickHandler.
func NewClickHandler(ref RefListener) *ClickHandler {
	return &ClickHandler{ref: ref}
}

// OnClick registers a handler for the "click" event.
func (h *ClickHandler) OnClick(handler func(ClickEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClickEvent(evt)) }
	h.ref.AddListener("click", wrapped, ClickEvent{}.props())
}

// OnDoubleClick registers a handler for the "dblclick" event.
func (h *ClickHandler) OnDoubleClick(handler func(ClickEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClickEvent(evt)) }
	h.ref.AddListener("dblclick", wrapped, ClickEvent{}.props())
}

// OnContextMenu registers a handler for the "contextmenu" event.
func (h *ClickHandler) OnContextMenu(handler func(ClickEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildClickEvent(evt)) }
	h.ref.AddListener("contextmenu", wrapped, ClickEvent{}.props())
}

func buildClickEvent(evt Event) ClickEvent {
	return ClickEvent{
		Event:    evt,
		Detail:   payloadInt(evt.Payload, "event.detail", 0),
		AltKey:   payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:  payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey: payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:  payloadBool(evt.Payload, "event.metaKey", false),
		Button:   payloadInt(evt.Payload, "event.button", 0),
		Buttons:  payloadInt(evt.Payload, "event.buttons", 0),
		ClientX:  payloadFloat(evt.Payload, "event.clientX", 0),
		ClientY:  payloadFloat(evt.Payload, "event.clientY", 0),
		OffsetX:  payloadFloat(evt.Payload, "event.offsetX", 0),
		OffsetY:  payloadFloat(evt.Payload, "event.offsetY", 0),
		PageX:    payloadFloat(evt.Payload, "event.pageX", 0),
		PageY:    payloadFloat(evt.Payload, "event.pageY", 0),
		ScreenX:  payloadFloat(evt.Payload, "event.screenX", 0),
		ScreenY:  payloadFloat(evt.Payload, "event.screenY", 0),
	}
}
