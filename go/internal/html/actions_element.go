package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ElementActions provides common DOM actions available on all elements.
// This is the base action mixin embedded in all ref types.
type ElementActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.ActionExecutor
}

func NewElementActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.ActionExecutor) *ElementActions[T] {
	return &ElementActions[T]{ref: ref, ctx: ctx}
}

// ScrollIntoView scrolls the element into view with the provided options.
func (a *ElementActions[T]) ScrollIntoView(opts dom.ScrollOptions) {
	dom.DOMScrollIntoView[T](a.ctx, a.ref, opts)
}

// SetScrollTop sets the scrollTop property of the element.
func (a *ElementActions[T]) SetScrollTop(value float64) {
	dom.DOMSet[T](a.ctx, a.ref, "scrollTop", value)
}

// SetScrollLeft sets the scrollLeft property of the element.
func (a *ElementActions[T]) SetScrollLeft(value float64) {
	dom.DOMSet[T](a.ctx, a.ref, "scrollLeft", value)
}

// Call invokes an arbitrary method on the element with the provided arguments.
// Use this as a fallback for methods not exposed as typed methods.
func (a *ElementActions[T]) Call(method string, args ...any) {
	dom.DOMCall[T](a.ctx, a.ref, method, args...)
}
