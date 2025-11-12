package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ElementActions provides common DOM actions available on all elements.
// This is the base action mixin embedded in all ref types.
type ElementActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewElementActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *ElementActions[T] {
	return &ElementActions[T]{ref: ref, ctx: ctx}
}

// Call invokes an arbitrary method on the element with the provided arguments.
// Use this as a fallback for methods not exposed as typed methods.
func (a *ElementActions[T]) Call(method string, args ...any) {
	dom.DOMCall[T](a.ctx, a.ref, method, args...)
}
