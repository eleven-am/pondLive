package html

// InputEvent represents input and change events on form elements.
type InputEvent struct {
	Event
	Value          string // Current value
	Checked        bool   // For checkboxes/radios
	SelectionStart int    // Text selection start
	SelectionEnd   int    // Text selection end
}

// Props returns the list of properties this event needs from the client.
func (InputEvent) props() []string {
	return []string{
		"target.value",
		"target.checked",
		"target.selectionStart",
		"target.selectionEnd",
	}
}

// InputHandler provides input event handlers.
type InputHandler struct {
	ref RefListener
}

// NewInputHandler creates a new InputHandler.
func NewInputHandler(ref RefListener) *InputHandler {
	return &InputHandler{ref: ref}
}

// OnInput registers a handler for the "input" event.
func (h *InputHandler) OnInput(handler func(InputEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildInputEvent(evt)) }
	h.ref.AddListener("input", wrapped, InputEvent{}.props())
}

// OnChange registers a handler for the "change" event.
func (h *InputHandler) OnChange(handler func(InputEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildInputEvent(evt)) }
	h.ref.AddListener("change", wrapped, InputEvent{}.props())
}

// OnSelect registers a handler for the "select" event.
func (h *InputHandler) OnSelect(handler func(InputEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildInputEvent(evt)) }
	h.ref.AddListener("select", wrapped, InputEvent{}.props())
}

func buildInputEvent(evt Event) InputEvent {
	return InputEvent{
		Event:          evt,
		Value:          payloadString(evt.Payload, "target.value", ""),
		Checked:        payloadBool(evt.Payload, "target.checked", false),
		SelectionStart: payloadInt(evt.Payload, "target.selectionStart", 0),
		SelectionEnd:   payloadInt(evt.Payload, "target.selectionEnd", 0),
	}
}
