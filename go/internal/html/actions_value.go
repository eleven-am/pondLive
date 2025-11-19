package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// ValueActions provides value-related DOM property setters for form elements.
// Embedded in refs for input, textarea, select, and other form controls.
type ValueActions[T dom2.ElementDescriptor] struct {
	ref *dom2.ElementRef[T]
	ctx dom2.Dispatcher
}

func NewValueActions[T dom2.ElementDescriptor](ref *dom2.ElementRef[T], ctx dom2.Dispatcher) *ValueActions[T] {
	return &ValueActions[T]{ref: ref, ctx: ctx}
}

// SetValue sets the value property of the element.
//
// Example:
//
//	inputRef := ui.UseElement[*h.InputRef](ctx)
//	inputRef.SetValue("New value")  // Update input value
//
//	return h.Input(h.Attach(inputRef), h.Type("text"))
func (a *ValueActions[T]) SetValue(value string) {
	dom2.DOMSet[T](a.ctx, a.ref, "value", value)
}

// SetChecked sets the checked property of the element (for checkboxes and radio buttons).
//
// Example:
//
//	checkboxRef := ui.UseElement[*h.InputRef](ctx)
//	checkboxRef.SetChecked(true)  // Check the checkbox
//
//	return h.Input(h.Attach(checkboxRef), h.Type("checkbox"))
func (a *ValueActions[T]) SetChecked(checked bool) {
	dom2.DOMSet[T](a.ctx, a.ref, "checked", checked)
}

// SetSelectedIndex sets the selectedIndex property of a select element.
//
// Example:
//
//	selectRef := ui.UseElement[*h.SelectRef](ctx)
//	selectRef.SetSelectedIndex(2)  // Select the third option (0-indexed)
//
//	return h.Select(h.Attach(selectRef), h.Children(
//	    h.Option(h.Text("Option 1")),
//	    h.Option(h.Text("Option 2")),
//	    h.Option(h.Text("Option 3")),
//	))
func (a *ValueActions[T]) SetSelectedIndex(index int) {
	dom2.DOMSet[T](a.ctx, a.ref, "selectedIndex", index)
}
