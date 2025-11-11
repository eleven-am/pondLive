package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ValueActions provides value-related DOM property setters for form elements.
// Embedded in refs for input, textarea, select, and other form controls.
type ValueActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
}

func NewValueActions[T dom.ElementDescriptor](ref *dom.ElementRef[T]) *ValueActions[T] {
	return &ValueActions[T]{ref: ref}
}

// SetValue sets the value property of the element.
func (a *ValueActions[T]) SetValue(ctx dom.ActionExecutor, value string) {
	dom.DOMSet[T](ctx, a.ref, "value", value)
}

// SetChecked sets the checked property of the element (for checkboxes and radio buttons).
func (a *ValueActions[T]) SetChecked(ctx dom.ActionExecutor, checked bool) {
	dom.DOMSet[T](ctx, a.ref, "checked", checked)
}

// SetDisabled sets the disabled property of the element.
func (a *ValueActions[T]) SetDisabled(ctx dom.ActionExecutor, disabled bool) {
	dom.DOMSet[T](ctx, a.ref, "disabled", disabled)
}

// SetSelectedIndex sets the selectedIndex property of a select element.
func (a *ValueActions[T]) SetSelectedIndex(ctx dom.ActionExecutor, index int) {
	dom.DOMSet[T](ctx, a.ref, "selectedIndex", index)
}
