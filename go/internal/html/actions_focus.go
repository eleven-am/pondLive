package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// FocusActions provides focus-related DOM actions for focusable elements.
// Embedded in refs for elements that support focus (input, textarea, button, select, a, etc.).
type FocusActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.ActionExecutor
}

func NewFocusActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.ActionExecutor) *FocusActions[T] {
	return &FocusActions[T]{ref: ref, ctx: ctx}
}

// Focus sets focus on the element.
func (a *FocusActions[T]) Focus() {
	dom.DOMCall[T](a.ctx, a.ref, "focus")
}

// Blur removes focus from the element.
func (a *FocusActions[T]) Blur() {
	dom.DOMCall[T](a.ctx, a.ref, "blur")
}
