package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// FormEvent represents form events (submit, reset, invalid).
type FormEvent struct {
	dom2.Event
}

// props returns the list of properties this event needs from the client.
func (FormEvent) props() []string {
	return []string{}
}

func buildFormEvent(evt dom2.Event) FormEvent {
	return FormEvent{
		Event: evt,
	}
}

// FormAPI provides actions and events for form elements.
type FormAPI[T dom2.ElementDescriptor] struct {
	ref *dom2.ElementRef[T]
	ctx dom2.Dispatcher
}

func NewFormAPI[T dom2.ElementDescriptor](ref *dom2.ElementRef[T], ctx dom2.Dispatcher) *FormAPI[T] {
	return &FormAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Submit submits the form programmatically, triggering validation and submission.
//
// Example:
//
//	formRef := ui.UseElement[*h.FormRef](ctx)
//	formRef.Submit()  // Programmatically submit the form
//
//	return h.Form(h.Attach(formRef), h.Action("/submit"))
func (a *FormAPI[T]) Submit() {
	dom2.DOMCall[T](a.ctx, a.ref, "submit")
}

// Reset resets all form controls to their initial values.
//
// Example:
//
//	formRef := ui.UseElement[*h.FormRef](ctx)
//	formRef.Reset()  // Clear all form fields
//
//	return h.Form(h.Attach(formRef), h.Action("/submit"))
func (a *FormAPI[T]) Reset() {
	dom2.DOMCall[T](a.ctx, a.ref, "reset")
}

// ============================================================================
// Events
// ============================================================================

// OnSubmit registers a handler for the "submit" event, fired when the form is submitted.
// Call evt.PreventDefault() to handle submission in your code instead of native browser behavior.
//
// Example:
//
//	formRef := ui.UseElement[*h.FormRef](ctx)
//	formRef.OnSubmit(func(evt h.FormEvent) h.Updates {
//	    evt.PreventDefault()  // Prevent native form submission
//	    handleFormSubmission()
//	    return nil
//	})
//
//	return h.Form(h.Attach(formRef), h.Action("/submit"))
func (a *FormAPI[T]) OnSubmit(handler func(FormEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildFormEvent(evt)) }
	a.ref.AddListener("submit", wrapped, FormEvent{}.props())
}

// OnReset registers a handler for the "reset" event, fired when the form is reset.
//
// Example:
//
//	formRef := ui.UseElement[*h.FormRef](ctx)
//	formRef.OnReset(func(evt h.FormEvent) h.Updates {
//	    clearValidationErrors()
//	    return nil
//	})
//
//	return h.Form(h.Attach(formRef), h.Action("/submit"))
func (a *FormAPI[T]) OnReset(handler func(FormEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildFormEvent(evt)) }
	a.ref.AddListener("reset", wrapped, FormEvent{}.props())
}

// OnInvalid registers a handler for the "invalid" event, fired when form validation fails.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.OnInvalid(func(evt h.FormEvent) h.Updates {
//	    showValidationError()
//	    return nil
//	})
//
//	return h.Input(h.Attach(inputRef), h.Required("true"))
func (a *FormAPI[T]) OnInvalid(handler func(FormEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildFormEvent(evt)) }
	a.ref.AddListener("invalid", wrapped, FormEvent{}.props())
}
