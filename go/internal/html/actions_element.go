package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// DOMRect represents the size and position of an element's bounding box.
type DOMRect struct {
	X      float64 // X coordinate of the element
	Y      float64 // Y coordinate of the element
	Width  float64 // Width of the element
	Height float64 // Height of the element
	Top    float64 // Top position relative to viewport
	Right  float64 // Right position relative to viewport
	Bottom float64 // Bottom position relative to viewport
	Left   float64 // Left position relative to viewport
}

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

// GetBoundingClientRect returns the size and position of the element.
// This makes a synchronous call to the client and waits for the response.
func (a *ElementActions[T]) GetBoundingClientRect() (*DOMRect, error) {
	result, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "getBoundingClientRect")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	rectMap, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	return &DOMRect{
		X:      payloadFloat(rectMap, "x", 0),
		Y:      payloadFloat(rectMap, "y", 0),
		Width:  payloadFloat(rectMap, "width", 0),
		Height: payloadFloat(rectMap, "height", 0),
		Top:    payloadFloat(rectMap, "top", 0),
		Right:  payloadFloat(rectMap, "right", 0),
		Bottom: payloadFloat(rectMap, "bottom", 0),
		Left:   payloadFloat(rectMap, "left", 0),
	}, nil
}
