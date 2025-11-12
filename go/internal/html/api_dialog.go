package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// DialogEvent represents dialog events (close, cancel).
type DialogEvent struct {
	Event
}

// props returns the list of properties this event needs from the client.
func (DialogEvent) props() []string {
	return []string{}
}

func buildDialogEvent(evt Event) DialogEvent {
	return DialogEvent{
		Event: evt,
	}
}

// DialogAPI provides actions and events for dialog elements.
type DialogAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewDialogAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *DialogAPI[T] {
	return &DialogAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Show displays the dialog non-modally.
func (a *DialogAPI[T]) Show() {
	dom.DOMCall[T](a.ctx, a.ref, "show")
}

// ShowModal displays the dialog modally (with backdrop and focus trap).
func (a *DialogAPI[T]) ShowModal() {
	dom.DOMCall[T](a.ctx, a.ref, "showModal")
}

// Close closes the dialog with an optional return value.
func (a *DialogAPI[T]) Close(returnValue string) {
	if returnValue == "" {
		dom.DOMCall[T](a.ctx, a.ref, "close")
	} else {
		dom.DOMCall[T](a.ctx, a.ref, "close", returnValue)
	}
}

// ============================================================================
// Events
// ============================================================================

// OnClose registers a handler for the "close" event.
func (a *DialogAPI[T]) OnClose(handler func(DialogEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDialogEvent(evt)) }
	a.ref.AddListener("close", wrapped, DialogEvent{}.props())
}

// OnCancel registers a handler for the "cancel" event.
func (a *DialogAPI[T]) OnCancel(handler func(DialogEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildDialogEvent(evt)) }
	a.ref.AddListener("cancel", wrapped, DialogEvent{}.props())
}
