package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// DisableableAPI provides actions for elements that can be disabled.
// This includes buttons and form controls (input, select, textarea, fieldset, etc).
type DisableableAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewDisableableAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *DisableableAPI[T] {
	return &DisableableAPI[T]{ref: ref, ctx: ctx}
}

// SetDisabled sets the disabled property of the element.
//
// Example:
//
//	buttonRef := ui.UseElement[*h.ButtonRef](ctx)
//	buttonRef.SetDisabled(true)  // Disable the button
//
//	return h.Button(h.Attach(buttonRef), h.Text("Submit"))
func (a *DisableableAPI[T]) SetDisabled(disabled bool) {
	dom.DOMSet[T](a.ctx, a.ref, "disabled", disabled)
}

// SetEnabled is a convenience method that sets disabled to false.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.SetEnabled(true)  // Enable the input field
//
//	return h.Input(h.Attach(inputRef), h.Type("text"))
func (a *DisableableAPI[T]) SetEnabled(enabled bool) {
	dom.DOMSet[T](a.ctx, a.ref, "disabled", !enabled)
}
