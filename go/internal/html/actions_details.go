package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// DetailsActions provides details-related DOM actions for details elements.
// Embedded in refs for details elements.
type DetailsActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewDetailsActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *DetailsActions[T] {
	return &DetailsActions[T]{ref: ref, ctx: ctx}
}

// SetOpen sets the open state of the details element.
func (a *DetailsActions[T]) SetOpen(open bool) {
	dom.DOMSet[T](a.ctx, a.ref, "open", open)
}
