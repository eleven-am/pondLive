package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// FormEvent represents form events (submit, reset, invalid).
type FormEvent struct {
	Event
}

// props returns the list of properties this event needs from the client.
func (FormEvent) props() []string {
	return []string{}
}

func buildFormEvent(evt Event) FormEvent {
	return FormEvent{
		Event: evt,
	}
}

// FormAPI provides actions and events for form elements.
type FormAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewFormAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *FormAPI[T] {
	return &FormAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Submit submits the form programmatically.
func (a *FormAPI[T]) Submit() {
	dom.DOMCall[T](a.ctx, a.ref, "submit")
}

// Reset resets all form controls to their initial values.
func (a *FormAPI[T]) Reset() {
	dom.DOMCall[T](a.ctx, a.ref, "reset")
}

// ============================================================================
// Events
// ============================================================================

// OnSubmit registers a handler for the "submit" event.
func (a *FormAPI[T]) OnSubmit(handler func(FormEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFormEvent(evt)) }
	a.ref.AddListener("submit", wrapped, FormEvent{}.props())
}

// OnReset registers a handler for the "reset" event.
func (a *FormAPI[T]) OnReset(handler func(FormEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFormEvent(evt)) }
	a.ref.AddListener("reset", wrapped, FormEvent{}.props())
}

// OnInvalid registers a handler for the "invalid" event.
func (a *FormAPI[T]) OnInvalid(handler func(FormEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildFormEvent(evt)) }
	a.ref.AddListener("invalid", wrapped, FormEvent{}.props())
}
