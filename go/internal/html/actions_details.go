package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// DetailsActions provides details-related DOM actions for details elements.
// Embedded in refs for details elements.
type DetailsActions[T dom2.ElementDescriptor] struct {
	ref *dom2.ElementRef[T]
	ctx dom2.Dispatcher
}

func NewDetailsActions[T dom2.ElementDescriptor](ref *dom2.ElementRef[T], ctx dom2.Dispatcher) *DetailsActions[T] {
	return &DetailsActions[T]{ref: ref, ctx: ctx}
}

// SetOpen sets the open state of the details element.
//
// Example:
//
//	detailsRef := ui.UseElement[*h.DetailsRef](ctx)
//	detailsRef.SetOpen(true)  // Expand the details element
//
//	return h.Details(h.Attach(detailsRef), h.Children(
//	    h.Summary(h.Text("Click to expand")),
//	    h.P(h.Text("Hidden content")),
//	))
func (a *DetailsActions[T]) SetOpen(open bool) {
	dom2.DOMSet[T](a.ctx, a.ref, "open", open)
}
