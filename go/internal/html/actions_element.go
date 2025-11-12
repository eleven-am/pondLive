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

// ScrollMetrics represents detailed scroll information for an element.
type ScrollMetrics struct {
	ScrollTop    float64 // Current vertical scroll position
	ScrollLeft   float64 // Current horizontal scroll position
	ScrollHeight float64 // Total scrollable height
	ScrollWidth  float64 // Total scrollable width
	ClientHeight float64 // Visible height (excluding scrollbar)
	ClientWidth  float64 // Visible width (excluding scrollbar)
}

// WindowMetrics represents viewport and scroll information for the window.
type WindowMetrics struct {
	InnerWidth  float64 // Viewport width
	InnerHeight float64 // Viewport height
	OuterWidth  float64 // Browser window width
	OuterHeight float64 // Browser window height
	ScrollX     float64 // Horizontal scroll position
	ScrollY     float64 // Vertical scroll position
	ScreenX     float64 // Window position on screen (X)
	ScreenY     float64 // Window position on screen (Y)
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

// GetScrollMetrics returns detailed scroll information for the element.
// This makes a synchronous call to the client and waits for the response.
func (a *ElementActions[T]) GetScrollMetrics() (*ScrollMetrics, error) {
	result, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "getScrollMetrics")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	metricsMap, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	return &ScrollMetrics{
		ScrollTop:    payloadFloat(metricsMap, "scrollTop", 0),
		ScrollLeft:   payloadFloat(metricsMap, "scrollLeft", 0),
		ScrollHeight: payloadFloat(metricsMap, "scrollHeight", 0),
		ScrollWidth:  payloadFloat(metricsMap, "scrollWidth", 0),
		ClientHeight: payloadFloat(metricsMap, "clientHeight", 0),
		ClientWidth:  payloadFloat(metricsMap, "clientWidth", 0),
	}, nil
}

// GetComputedStyle returns the computed CSS styles for the element.
// If no properties are specified, returns a commonly used subset of styles.
// This makes a synchronous call to the client and waits for the response.
func (a *ElementActions[T]) GetComputedStyle(properties ...string) (map[string]string, error) {
	result, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "getComputedStyle", properties)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	stylesMap, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	styles := make(map[string]string, len(stylesMap))
	for key := range stylesMap {
		styles[key] = PayloadString(stylesMap, key, "")
	}
	return styles, nil
}

// IsVisible checks if the element is visible in the viewport.
// This makes a synchronous call to the client and waits for the response.
func (a *ElementActions[T]) IsVisible() (bool, error) {
	result, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "checkVisibility")
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, nil
	}

	visible, ok := result.(bool)
	if !ok {
		return false, nil
	}
	return visible, nil
}

// Matches checks if the element matches the given CSS selector.
// This makes a synchronous call to the client and waits for the response.
func (a *ElementActions[T]) Matches(selector string) (bool, error) {
	result, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "matches", selector)
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, nil
	}

	matches, ok := result.(bool)
	if !ok {
		return false, nil
	}
	return matches, nil
}
