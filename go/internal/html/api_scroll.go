package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// ScrollEvent represents scroll events.
type ScrollEvent struct {
	Event
	ScrollTop    float64 // Current vertical scroll position
	ScrollLeft   float64 // Current horizontal scroll position
	ScrollHeight float64 // Total scrollable height
	ScrollWidth  float64 // Total scrollable width
	ClientHeight float64 // Visible height
	ClientWidth  float64 // Visible width
}

// props returns the list of properties this event needs from the client.
func (ScrollEvent) props() []string {
	return []string{
		"target.scrollTop",
		"target.scrollLeft",
		"target.scrollHeight",
		"target.scrollWidth",
		"target.clientHeight",
		"target.clientWidth",
	}
}

func buildScrollEvent(evt Event) ScrollEvent {
	return ScrollEvent{
		Event:        evt,
		ScrollTop:    payloadFloat(evt.Payload, "target.scrollTop", 0),
		ScrollLeft:   payloadFloat(evt.Payload, "target.scrollLeft", 0),
		ScrollHeight: payloadFloat(evt.Payload, "target.scrollHeight", 0),
		ScrollWidth:  payloadFloat(evt.Payload, "target.scrollWidth", 0),
		ClientHeight: payloadFloat(evt.Payload, "target.clientHeight", 0),
		ClientWidth:  payloadFloat(evt.Payload, "target.clientWidth", 0),
	}
}

// ScrollAPI provides actions and events for scrollable elements.
type ScrollAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewScrollAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *ScrollAPI[T] {
	return &ScrollAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// ScrollIntoView scrolls the element into view with the provided options.
//
// Example:
//
//	divRef := ui.UseElement[*h.DivRef](ctx)
//	divRef.ScrollIntoView(dom.ScrollOptions{Behavior: "smooth", Block: "center"})
//
//	return h.Div(h.Attach(divRef), h.Text("Scroll to me"))
func (a *ScrollAPI[T]) ScrollIntoView(opts dom.ScrollOptions) {
	dom.DOMScrollIntoView[T](a.ctx, a.ref, opts)
}

// SetScrollTop sets the vertical scroll position of the element.
//
// Example:
//
//	divRef := ui.UseElement[*h.DivRef](ctx)
//	divRef.SetScrollTop(100.0)  // Scroll down 100 pixels
//
//	return h.Div(h.Attach(divRef), h.Text("Scrollable content"))
func (a *ScrollAPI[T]) SetScrollTop(value float64) {
	dom.DOMSet[T](a.ctx, a.ref, "scrollTop", value)
}

// SetScrollLeft sets the horizontal scroll position of the element.
//
// Example:
//
//	divRef := ui.UseElement[*h.DivRef](ctx)
//	divRef.SetScrollLeft(50.0)  // Scroll right 50 pixels
//
//	return h.Div(h.Attach(divRef), h.Text("Scrollable content"))
func (a *ScrollAPI[T]) SetScrollLeft(value float64) {
	dom.DOMSet[T](a.ctx, a.ref, "scrollLeft", value)
}

// ============================================================================
// Events
// ============================================================================

// OnScroll registers a handler for the "scroll" event, fired when the element is scrolled.
//
// Example:
//
//	divRef := ui.UseElement[*h.DivRef](ctx)
//	divRef.OnScroll(func(evt h.ScrollEvent) h.Updates {
//	    updateScrollIndicator(evt.ScrollTop, evt.ScrollHeight, evt.ClientHeight)
//	    return nil
//	})
//
//	return h.Div(h.Attach(divRef), h.Text("Scrollable content"))
func (a *ScrollAPI[T]) OnScroll(handler func(ScrollEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildScrollEvent(evt)) }
	a.ref.AddListener("scroll", wrapped, ScrollEvent{}.props())
}
