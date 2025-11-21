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

// ElementActions provides common DOM actions and queries available on all HTML elements.
// This is the base action mixin embedded in all element ref types, offering fundamental
// DOM inspection capabilities like bounds checking, scroll metrics, and computed styles.
//
// Example:
//
//	divRef := ui.UseElement[*h.DivRef](ctx)
//
//	// Query element dimensions
//	rect, _ := divRef.GetBoundingClientRect()
//	fmt.Printf("Element at (%f, %f) with size %fx%f\n", rect.X, rect.Y, rect.Width, rect.Height)
//
//	return h.Div(h.Attach(divRef), h.Text("Container"))
type ElementActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewElementActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *ElementActions[T] {
	return &ElementActions[T]{ref: ref, ctx: ctx}
}

// Call invokes an arbitrary method on the DOM element with the provided arguments.
// This is a low-level escape hatch for calling any DOM method not exposed through typed APIs.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//
//	// Call a method not available in MediaAPI
//	videoRef.Call("requestPictureInPicture")
//
//	// Call with arguments
//	videoRef.Call("scrollTo", 0, 100)
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
//
// Note: This method provides no type safety. Use typed API methods when available.
// Arguments are serialized to JSON and sent to the client, so ensure they're JSON-serializable.
func (a *ElementActions[T]) Call(method string, args ...any) {
	dom.DOMCall[T](a.ctx, a.ref, method, args...)
}

// GetBoundingClientRect returns the size and position of the element relative to the viewport.
// This makes a synchronous call to the client (~1-2ms latency) and waits for the response.
//
// DOMRect fields:
//   - X, Y: Position relative to viewport top-left corner
//   - Width, Height: Element dimensions including padding and border
//   - Top, Right, Bottom, Left: Edges relative to viewport
//
// Example:
//
//	tooltipRef := ui.UseElement[*h.DivRef](ctx)
//
//	// Position tooltip relative to trigger element
//	triggerRect, err := tooltipRef.GetBoundingClientRect()
//	if err == nil {
//	    tooltipX := triggerRect.Right + 10 // 10px to the right
//	    tooltipY := triggerRect.Top
//	    positionTooltip(tooltipX, tooltipY)
//	}
//
//	return h.Div(h.Attach(tooltipRef), h.Text("Tooltip"))
//
// Use Cases:
//   - Positioning popups, tooltips, or dropdowns
//   - Detecting element visibility in viewport
//   - Calculating relative positions between elements
//   - Implementing drag-and-drop with precise positioning
//
// Note: Coordinates are relative to the viewport, not the document. For document-relative
// positions, add window scroll offsets (scrollX, scrollY) to the returned values.
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

// GetScrollMetrics returns detailed scroll information for scrollable elements.
// This makes multiple synchronous calls (~6-12ms total) to gather comprehensive scroll state.
//
// ScrollMetrics fields:
//   - ScrollTop/ScrollLeft: Current scroll position
//   - ScrollHeight/ScrollWidth: Total scrollable content size
//   - ClientHeight/ClientWidth: Visible area size (excluding scrollbar)
//
// Example:
//
//	chatRef := ui.UseElement[*h.DivRef](ctx)
//
//	// Check if user has scrolled to bottom
//	metrics, err := chatRef.GetScrollMetrics()
//	if err == nil {
//	    atBottom := metrics.ScrollTop + metrics.ClientHeight >= metrics.ScrollHeight - 10
//	    if atBottom {
//	        markAllMessagesAsRead()
//	    }
//
//	    // Calculate scroll percentage
//	    scrollPercent := (metrics.ScrollTop / (metrics.ScrollHeight - metrics.ClientHeight)) * 100
//	}
//
//	return h.Div(h.Attach(chatRef), h.Text("Chat window"))
//
// Use Cases:
//   - Implementing infinite scroll (detect when near bottom)
//   - Auto-scrolling to latest content
//   - Showing scroll position indicators
//   - Detecting scroll progress through long content
//   - Implementing "scroll to top" buttons
//
// Note: For non-scrollable elements, ScrollHeight/Width equals ClientHeight/Width and
// ScrollTop/Left will be 0. Use this to detect if an element is scrollable.
func (a *ElementActions[T]) GetScrollMetrics() (*ScrollMetrics, error) {
	scrollTop, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "scrollTop")
	if err != nil {
		return nil, err
	}
	scrollLeft, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "scrollLeft")
	if err != nil {
		return nil, err
	}
	scrollHeight, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "scrollHeight")
	if err != nil {
		return nil, err
	}
	scrollWidth, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "scrollWidth")
	if err != nil {
		return nil, err
	}
	clientHeight, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "clientHeight")
	if err != nil {
		return nil, err
	}
	clientWidth, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "clientWidth")
	if err != nil {
		return nil, err
	}

	return &ScrollMetrics{
		ScrollTop:    payloadFloatDirect(scrollTop, 0),
		ScrollLeft:   payloadFloatDirect(scrollLeft, 0),
		ScrollHeight: payloadFloatDirect(scrollHeight, 0),
		ScrollWidth:  payloadFloatDirect(scrollWidth, 0),
		ClientHeight: payloadFloatDirect(clientHeight, 0),
		ClientWidth:  payloadFloatDirect(clientWidth, 0),
	}, nil
}

// payloadFloatDirect converts a direct any value to float64
func payloadFloatDirect(value any, defaultValue float64) float64 {
	if value == nil {
		return defaultValue
	}
	if v, ok := value.(float64); ok {
		return v
	}
	return defaultValue
}

// GetComputedStyle returns the computed CSS styles for the element as resolved by the browser.
// If properties are specified, returns only those properties. Otherwise returns common styles.
// This makes a synchronous call (~1-2ms latency) to the client.
//
// Example:
//
//	buttonRef := ui.UseElement[*h.ButtonRef](ctx)
//
//	// Get specific styles
//	styles, err := buttonRef.GetComputedStyle("color", "backgroundColor", "fontSize")
//	if err == nil {
//	    textColor := styles["color"]          // e.g., "rgb(255, 255, 255)"
//	    bgColor := styles["backgroundColor"]  // e.g., "rgb(0, 123, 255)"
//	    fontSize := styles["fontSize"]        // e.g., "16px"
//	}
//
//	// Get all common styles (when no properties specified)
//	allStyles, _ := buttonRef.GetComputedStyle()
//
//	return h.Button(h.Attach(buttonRef), h.Text("Click me"))
//
// Use Cases:
//   - Reading theme colors or dimensions set by CSS
//   - Checking if element is hidden via display:none or visibility:hidden
//   - Getting animation/transition properties
//   - Debugging style inheritance issues
//   - Implementing responsive behavior based on computed styles
//
// Note: Computed styles are the final values after CSS cascade, inheritance, and defaults.
// Use camelCase for multi-word properties (e.g., "backgroundColor" not "background-color").
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

// CheckVisibility checks if the element is currently visible according to CSS visibility rules.
// This considers opacity, visibility, display, and content-visibility properties.
// This makes a synchronous call (~1-2ms latency) to the client.
//
// Example:
//
//	modalRef := ui.UseElement[*h.DivRef](ctx)
//
//	// Check if modal is visible before showing toast
//	visible, err := modalRef.CheckVisibility()
//	if err == nil && !visible {
//	    showToastNotification()
//	}
//
//	return h.Div(h.Attach(modalRef), h.Text("Modal"))
//
// Use Cases:
//   - Conditional logic based on visibility state
//   - Verifying animations completed correctly
//   - Implementing focus management (don't focus hidden elements)
//   - A/B testing visibility experiments
//
// Note: This checks CSS-level visibility, not whether the element is in the viewport.
// An element can be "visible" according to this method but scrolled out of view.
// Use GetBoundingClientRect() to check viewport visibility.
func (a *ElementActions[T]) CheckVisibility() (bool, error) {
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
// This is useful for dynamic behavior based on element state or attributes.
// This makes a synchronous call (~1-2ms latency) to the client.
//
// Example:
//
//	listItemRef := ui.UseElement[*h.LiRef](ctx)
//
//	// Check if element has specific class or pseudo-class
//	isActive, _ := listItemRef.Matches(".active")
//	isFirst, _ := listItemRef.Matches(":first-child")
//	isChecked, _ := listItemRef.Matches(":checked")
//
//	// Complex selectors work too
//	matches, _ := listItemRef.Matches("li.item[data-status='completed']:not(.archived)")
//	if matches {
//	    // Element matches all criteria
//	}
//
//	return h.Li(h.Attach(listItemRef), h.Text("Item"))
//
// Use Cases:
//   - Checking if element has specific classes or attributes
//   - Testing pseudo-classes (:hover, :focus, :checked, etc.)
//   - Validating element state matches expected selector
//   - Implementing conditional logic based on CSS selectors
//
// Note: This is equivalent to element.matches(selector) in JavaScript. The selector is evaluated
// in the browser, so all standard CSS selectors including pseudo-classes are supported.
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
