package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// SelectionActions provides text selection DOM actions for text input elements.
// Embedded in refs for input and textarea elements.
type SelectionActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.ActionExecutor
}

func NewSelectionActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.ActionExecutor) *SelectionActions[T] {
	return &SelectionActions[T]{ref: ref, ctx: ctx}
}

// Select selects all text in the element.
func (a *SelectionActions[T]) Select() {
	dom.DOMCall[T](a.ctx, a.ref, "select")
}

// SetSelectionRange sets the selection range for the text in the element.
func (a *SelectionActions[T]) SetSelectionRange(start, end int) {
	dom.DOMCall[T](a.ctx, a.ref, "setSelectionRange", start, end)
}
