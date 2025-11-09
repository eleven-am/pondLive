package html

import "github.com/eleven-am/pondlive/go/internal/dom"

type (
	RefListener = dom.RefListener
)

type ElementRef[T ElementDescriptor] struct {
	*dom.ElementRef[T]
}

func (r *ElementRef[T]) Ref() *ElementRef[T] {
	if r == nil {
		return nil
	}
	return r
}

func (r *ElementRef[T]) DOMElementRef() *dom.ElementRef[T] {
	if r == nil {
		return nil
	}
	return r.ElementRef
}

type AttachTarget[T ElementDescriptor] interface {
	Ref() *ElementRef[T]
}

func NewElementRef[T ElementDescriptor](id string, descriptor T) *ElementRef[T] {
	raw := dom.NewElementRef(id, descriptor)
	if raw == nil {
		return nil
	}
	return &ElementRef[T]{ElementRef: raw}
}

func Attach[T ElementDescriptor](target AttachTarget[T]) Prop {
	ref := target.Ref()
	if ref == nil {
		return nil
	}
	return elementAttachProp[T]{ref: ref}
}

type elementAttachProp[T ElementDescriptor] struct {
	ref *ElementRef[T]
}

func (elementAttachProp[T]) isProp() {}

func (p elementAttachProp[T]) ApplyTo(e *Element) {
	dom.AttachElementRef[T](p.ref.ElementRef, e)
}

// no builder registry needed; ref construction handled in generated façade
