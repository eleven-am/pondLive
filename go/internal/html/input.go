package html

import "github.com/eleven-am/pondlive/go/internal/dom"

// InputEvent represents input and change events on form elements.
type InputEvent struct {
	dom.Event
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
	ref dom.RefListener
}

// NewInputHandler creates a new InputHandler.
func NewInputHandler(ref dom.RefListener) *InputHandler {
	return &InputHandler{ref: ref}
}

// OnInput registers a handler for the "input" event.
func (h *InputHandler) OnInput(handler func(InputEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildInputEvent(evt)) }
	h.ref.AddListener("input", wrapped, InputEvent{}.props())
}

// OnChange registers a handler for the "change" event.
func (h *InputHandler) OnChange(handler func(InputEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildInputEvent(evt)) }
	h.ref.AddListener("change", wrapped, InputEvent{}.props())
}

// OnSelect registers a handler for the "select" event.
func (h *InputHandler) OnSelect(handler func(InputEvent) dom.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom.Event) dom.Updates { return handler(buildInputEvent(evt)) }
	h.ref.AddListener("select", wrapped, InputEvent{}.props())
}

func buildInputEvent(evt dom.Event) InputEvent {
	detail := extractDetail(evt.Payload)
	return InputEvent{
		Event:          evt,
		Value:          PayloadString(detail, "target.value", ""),
		Checked:        payloadBool(detail, "target.checked", false),
		SelectionStart: payloadInt(detail, "target.selectionStart", 0),
		SelectionEnd:   payloadInt(detail, "target.selectionEnd", 0),
	}
}
