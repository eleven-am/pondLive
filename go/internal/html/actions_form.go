package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// FormActions provides form-related DOM actions for form elements.
// Embedded in refs for form elements.
type FormActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.ActionExecutor
}

func NewFormActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.ActionExecutor) *FormActions[T] {
	return &FormActions[T]{ref: ref, ctx: ctx}
}

// Submit submits the form programmatically.
func (a *FormActions[T]) Submit() {
	dom.DOMCall[T](a.ctx, a.ref, "submit")
}

// Reset resets all form controls to their initial values.
func (a *FormActions[T]) Reset() {
	dom.DOMCall[T](a.ctx, a.ref, "reset")
}
