package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// DetailsActions provides details-related DOM actions for details elements.
// Embedded in refs for details elements.
type DetailsActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
}

func NewDetailsActions[T dom.ElementDescriptor](ref *dom.ElementRef[T]) *DetailsActions[T] {
	return &DetailsActions[T]{ref: ref}
}

// SetOpen sets the open state of the details element.
func (a *DetailsActions[T]) SetOpen(ctx dom.ActionExecutor, open bool) {
	dom.DOMSet[T](ctx, a.ref, "open", open)
}
