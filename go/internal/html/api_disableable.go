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
func (a *DisableableAPI[T]) SetDisabled(disabled bool) {
	dom.DOMSet[T](a.ctx, a.ref, "disabled", disabled)
}

// SetEnabled is a convenience method that sets disabled to false.
func (a *DisableableAPI[T]) SetEnabled(enabled bool) {
	dom.DOMSet[T](a.ctx, a.ref, "disabled", !enabled)
}
