package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// SelectionActions provides text selection DOM actions for text input elements.
// Embedded in refs for input and textarea elements.
type SelectionActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
}

func NewSelectionActions[T dom.ElementDescriptor](ref *dom.ElementRef[T]) *SelectionActions[T] {
	return &SelectionActions[T]{ref: ref}
}

// Select selects all text in the element.
func (a *SelectionActions[T]) Select(ctx dom.ActionExecutor) {
	dom.DOMCall[T](ctx, a.ref, "select")
}

// SetSelectionRange sets the selection range for the text in the element.
func (a *SelectionActions[T]) SetSelectionRange(ctx dom.ActionExecutor, start, end int) {
	dom.DOMCall[T](ctx, a.ref, "setSelectionRange", start, end)
}
