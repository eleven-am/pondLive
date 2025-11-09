package html

import "github.com/eleven-am/pondlive/go/internal/dom"

type ElementHookContext interface{}

func useElementRef[T ElementDescriptor](ctx ElementHookContext) *ElementRef[T] {
	if ctx == nil {
		return nil
	}
	var descriptor T
	raw := dom.AcquireElementRef(ctx, descriptor)
	if raw == nil {
		return nil
	}
	validated := dom.ValidateElementRef(raw, descriptor)
	if validated == nil {
		return nil
	}
	return &ElementRef[T]{ElementRef: validated}
}
