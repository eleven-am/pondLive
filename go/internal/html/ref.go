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

func (r *ElementRef[T]) AttachTo(e *Element) {
	if r == nil {
		return
	}
	dom.AttachElementRef[T](r.ElementRef, e)
}

func NewElementRef[T ElementDescriptor](id string, descriptor T) *ElementRef[T] {
	raw := dom.NewElementRef(id, descriptor)
	if raw == nil {
		return nil
	}
	return &ElementRef[T]{ElementRef: raw}
}

type Attachment interface {
	AttachTo(*Element)
}

func Attach(target Attachment) Prop {
	if target == nil {
		return nil
	}
	return attachmentProp{target: target}
}

type attachmentProp struct {
	target Attachment
}

func (attachmentProp) isProp() {}

func (p attachmentProp) ApplyTo(e *Element) {
	if e == nil || p.target == nil {
		return
	}
	p.target.AttachTo(e)
}

// no builder registry needed; ref construction handled in generated façade
