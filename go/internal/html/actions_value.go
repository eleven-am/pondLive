package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ValueActions provides value-related DOM property setters for form elements.
// Embedded in refs for input, textarea, select, and other form controls.
type ValueActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewValueActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *ValueActions[T] {
	return &ValueActions[T]{ref: ref, ctx: ctx}
}

// SetValue sets the value property of the element.
func (a *ValueActions[T]) SetValue(value string) {
	dom.DOMSet[T](a.ctx, a.ref, "value", value)
}

// SetChecked sets the checked property of the element (for checkboxes and radio buttons).
func (a *ValueActions[T]) SetChecked(checked bool) {
	dom.DOMSet[T](a.ctx, a.ref, "checked", checked)
}

// SetSelectedIndex sets the selectedIndex property of a select element.
func (a *ValueActions[T]) SetSelectedIndex(index int) {
	dom.DOMSet[T](a.ctx, a.ref, "selectedIndex", index)
}
