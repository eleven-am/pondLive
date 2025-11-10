package html

// KeyboardEvent represents keyboard events (keydown, keyup, keypress).
type KeyboardEvent struct {
	Event
	Key         string // The key value
	Code        string // Physical key code
	Location    int    // Location of the key on keyboard
	Repeat      bool   // Is key being held down
	AltKey      bool   // Alt key pressed
	CtrlKey     bool   // Control key pressed
	ShiftKey    bool   // Shift key pressed
	MetaKey     bool   // Meta key pressed
	IsComposing bool   // Is part of composition
}

// Props returns the list of properties this event needs from the client.
func (KeyboardEvent) props() []string {
	return []string{
		"event.key",
		"event.code",
		"event.location",
		"event.repeat",
		"event.altKey",
		"event.ctrlKey",
		"event.shiftKey",
		"event.metaKey",
		"event.isComposing",
	}
}

// KeyboardHandler provides keyboard event handlers.
type KeyboardHandler struct {
	ref RefListener
}

// NewKeyboardHandler creates a new KeyboardHandler.
func NewKeyboardHandler(ref RefListener) *KeyboardHandler {
	return &KeyboardHandler{ref: ref}
}

// OnKeyDown registers a handler for the "keydown" event.
func (h *KeyboardHandler) OnKeyDown(handler func(KeyboardEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildKeyboardEvent(evt)) }
	h.ref.AddListener("keydown", wrapped, KeyboardEvent{}.props())
}

// OnKeyUp registers a handler for the "keyup" event.
func (h *KeyboardHandler) OnKeyUp(handler func(KeyboardEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildKeyboardEvent(evt)) }
	h.ref.AddListener("keyup", wrapped, KeyboardEvent{}.props())
}

// OnKeyPress registers a handler for the "keypress" event.
func (h *KeyboardHandler) OnKeyPress(handler func(KeyboardEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildKeyboardEvent(evt)) }
	h.ref.AddListener("keypress", wrapped, KeyboardEvent{}.props())
}

func buildKeyboardEvent(evt Event) KeyboardEvent {
	return KeyboardEvent{
		Event:       evt,
		Key:         PayloadString(evt.Payload, "event.key", ""),
		Code:        PayloadString(evt.Payload, "event.code", ""),
		Location:    payloadInt(evt.Payload, "event.location", 0),
		Repeat:      payloadBool(evt.Payload, "event.repeat", false),
		AltKey:      payloadBool(evt.Payload, "event.altKey", false),
		CtrlKey:     payloadBool(evt.Payload, "event.ctrlKey", false),
		ShiftKey:    payloadBool(evt.Payload, "event.shiftKey", false),
		MetaKey:     payloadBool(evt.Payload, "event.metaKey", false),
		IsComposing: payloadBool(evt.Payload, "event.isComposing", false),
	}
}
