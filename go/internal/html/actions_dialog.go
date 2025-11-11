package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// DialogActions provides dialog-related DOM actions for dialog elements.
// Embedded in refs for dialog elements.
type DialogActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
}

func NewDialogActions[T dom.ElementDescriptor](ref *dom.ElementRef[T]) *DialogActions[T] {
	return &DialogActions[T]{ref: ref}
}

// Show displays the dialog non-modally.
func (a *DialogActions[T]) Show(ctx dom.ActionExecutor) {
	dom.DOMCall[T](ctx, a.ref, "show")
}

// ShowModal displays the dialog modally (with backdrop and focus trap).
func (a *DialogActions[T]) ShowModal(ctx dom.ActionExecutor) {
	dom.DOMCall[T](ctx, a.ref, "showModal")
}

// Close closes the dialog with an optional return value.
func (a *DialogActions[T]) Close(ctx dom.ActionExecutor, returnValue string) {
	if returnValue == "" {
		dom.DOMCall[T](ctx, a.ref, "close")
	} else {
		dom.DOMCall[T](ctx, a.ref, "close", returnValue)
	}
}
