package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ElementActions provides common DOM actions available on all elements.
// This is the base action mixin embedded in all ref types.
type ElementActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
}

func NewElementActions[T dom.ElementDescriptor](ref *dom.ElementRef[T]) *ElementActions[T] {
	return &ElementActions[T]{ref: ref}
}

// ScrollIntoView scrolls the element into view with the provided options.
func (a *ElementActions[T]) ScrollIntoView(ctx dom.ActionExecutor, opts dom.ScrollOptions) {
	dom.DOMScrollIntoView[T](ctx, a.ref, opts)
}

// SetScrollTop sets the scrollTop property of the element.
func (a *ElementActions[T]) SetScrollTop(ctx dom.ActionExecutor, value float64) {
	dom.DOMSet[T](ctx, a.ref, "scrollTop", value)
}

// SetScrollLeft sets the scrollLeft property of the element.
func (a *ElementActions[T]) SetScrollLeft(ctx dom.ActionExecutor, value float64) {
	dom.DOMSet[T](ctx, a.ref, "scrollLeft", value)
}

// Call invokes an arbitrary method on the element with the provided arguments.
// Use this as a fallback for methods not exposed as typed methods.
func (a *ElementActions[T]) Call(ctx dom.ActionExecutor, method string, args ...any) {
	dom.DOMCall[T](ctx, a.ref, method, args...)
}
