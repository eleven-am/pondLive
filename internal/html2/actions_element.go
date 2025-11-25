package html2

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
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

// ElementActions provides common DOM actions and queries available on all HTML elements.
// This is the base action mixin embedded in all element action types.
//
// Example:
//
//	divRef := ui.UseRef(ctx)
//	actions := html.Element(ctx, divRef)
//
//	rect, _ := actions.GetBoundingClientRect()
//	fmt.Printf("Element at (%f, %f)\n", rect.X, rect.Y)
//
//	return html.El("div", html.Attach(divRef), html.Text("Container"))
type ElementActions struct {
	ctx *runtime2.Ctx
	ref work.Attachment
}

// NewElementActions creates an ElementActions for the given ref.
func NewElementActions(ctx *runtime2.Ctx, ref work.Attachment) *ElementActions {
	return &ElementActions{ctx: ctx, ref: ref}
}

// Call invokes an arbitrary method on the DOM element with the provided arguments.
// This is a low-level escape hatch for calling any DOM method not exposed through typed APIs.
//
// Example:
//
//	actions := html.Element(ctx, ref)
//	actions.Call("scrollTo", 0, 100)
func (a *ElementActions) Call(method string, args ...any) {
	if a.ctx == nil || a.ref == nil {
		return
	}
	a.ctx.Call(a.ref, method, args...)
}

// Set assigns a value to a property on the element.
func (a *ElementActions) Set(prop string, value any) {
	if a.ctx == nil || a.ref == nil {
		return
	}
	a.ctx.Set(a.ref, prop, value)
}

// Query retrieves property values from the element.
func (a *ElementActions) Query(selectors ...string) (map[string]any, error) {
	if a.ctx == nil || a.ref == nil {
		return nil, runtime2.ErrNilRef
	}
	return a.ctx.Query(a.ref, selectors...)
}

// AsyncCall invokes a method and waits for the result.
func (a *ElementActions) AsyncCall(method string, args ...any) (any, error) {
	if a.ctx == nil || a.ref == nil {
		return nil, runtime2.ErrNilRef
	}
	return a.ctx.AsyncCall(a.ref, method, args...)
}

// Focus sets focus on the element.
func (a *ElementActions) Focus() {
	a.Call("focus")
}

// Blur removes focus from the element.
func (a *ElementActions) Blur() {
	a.Call("blur")
}

// Click simulates a click on the element.
func (a *ElementActions) Click() {
	a.Call("click")
}

// GetBoundingClientRect returns the size and position of the element relative to the viewport.
// This makes a synchronous call to the client and waits for the response.
//
// Example:
//
//	actions := html.Element(ctx, ref)
//	rect, err := actions.GetBoundingClientRect()
//	if err == nil {
//	    tooltipX := rect.Right + 10
//	    tooltipY := rect.Top
//	}
func (a *ElementActions) GetBoundingClientRect() (*DOMRect, error) {
	result, err := a.AsyncCall("getBoundingClientRect")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	m, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	return &DOMRect{
		X:      toFloat64(m["x"]),
		Y:      toFloat64(m["y"]),
		Width:  toFloat64(m["width"]),
		Height: toFloat64(m["height"]),
		Top:    toFloat64(m["top"]),
		Right:  toFloat64(m["right"]),
		Bottom: toFloat64(m["bottom"]),
		Left:   toFloat64(m["left"]),
	}, nil
}

// GetScrollMetrics returns detailed scroll information for scrollable elements.
// This makes a synchronous call to the client and waits for the response.
//
// Example:
//
//	actions := html.Element(ctx, ref)
//	metrics, err := actions.GetScrollMetrics()
//	if err == nil {
//	    atBottom := metrics.ScrollTop + metrics.ClientHeight >= metrics.ScrollHeight - 10
//	}
func (a *ElementActions) GetScrollMetrics() (*ScrollMetrics, error) {
	values, err := a.Query("scrollTop", "scrollLeft", "scrollHeight", "scrollWidth", "clientHeight", "clientWidth")
	if err != nil {
		return nil, err
	}
	return &ScrollMetrics{
		ScrollTop:    toFloat64(values["scrollTop"]),
		ScrollLeft:   toFloat64(values["scrollLeft"]),
		ScrollHeight: toFloat64(values["scrollHeight"]),
		ScrollWidth:  toFloat64(values["scrollWidth"]),
		ClientHeight: toFloat64(values["clientHeight"]),
		ClientWidth:  toFloat64(values["clientWidth"]),
	}, nil
}

// GetComputedStyle returns the computed CSS styles for the element.
// If properties are specified, returns only those properties.
//
// Example:
//
//	actions := html.Element(ctx, ref)
//	styles, err := actions.GetComputedStyle("color", "backgroundColor")
//	if err == nil {
//	    textColor := styles["color"]
//	}
func (a *ElementActions) GetComputedStyle(properties ...string) (map[string]string, error) {
	result, err := a.AsyncCall("getComputedStyle", properties)
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
	for key, val := range stylesMap {
		if s, ok := val.(string); ok {
			styles[key] = s
		}
	}
	return styles, nil
}

// CheckVisibility checks if the element is currently visible according to CSS visibility rules.
// This considers opacity, visibility, display, and content-visibility properties.
//
// Example:
//
//	actions := html.Element(ctx, ref)
//	visible, err := actions.CheckVisibility()
//	if err == nil && !visible {
//	    showNotification()
//	}
func (a *ElementActions) CheckVisibility() (bool, error) {
	result, err := a.AsyncCall("checkVisibility")
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

// Matches checks if the element matches the given CSS selector.
//
// Example:
//
//	actions := html.Element(ctx, ref)
//	isActive, _ := actions.Matches(".active")
//	isFirst, _ := actions.Matches(":first-child")
func (a *ElementActions) Matches(selector string) (bool, error) {
	result, err := a.AsyncCall("matches", selector)
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

// Helper functions for type conversion

func toFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

// ============================================================================
// Event Handler Methods
// ============================================================================

// addHandler registers an event handler on the ref if it supports adding handlers.
func (a *ElementActions) addHandler(event string, handler work.Handler) {
	if a.ref == nil {
		return
	}
	if adder, ok := a.ref.(work.HandlerAdder); ok {
		adder.AddHandler(event, handler)
	}
}

// ============================================================================
// Click Events
// ============================================================================

// OnClick registers a handler for click events.
func (a *ElementActions) OnClick(handler func(ClickEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("click", work.Handler{
		EventOptions: metadata.EventOptions{Props: ClickEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildClickEvent(evt)) },
	})
	return a
}

// OnDoubleClick registers a handler for double-click events.
func (a *ElementActions) OnDoubleClick(handler func(ClickEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("dblclick", work.Handler{
		EventOptions: metadata.EventOptions{Props: ClickEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildClickEvent(evt)) },
	})
	return a
}

// OnContextMenu registers a handler for right-click/context menu events.
func (a *ElementActions) OnContextMenu(handler func(ClickEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("contextmenu", work.Handler{
		EventOptions: metadata.EventOptions{Props: ClickEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildClickEvent(evt)) },
	})
	return a
}

// ============================================================================
// Mouse Events
// ============================================================================

// OnMouseDown registers a handler for mousedown events.
func (a *ElementActions) OnMouseDown(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mousedown", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// OnMouseUp registers a handler for mouseup events.
func (a *ElementActions) OnMouseUp(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mouseup", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// OnMouseMove registers a handler for mousemove events.
func (a *ElementActions) OnMouseMove(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mousemove", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// OnMouseEnter registers a handler for mouseenter events.
func (a *ElementActions) OnMouseEnter(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mouseenter", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// OnMouseLeave registers a handler for mouseleave events.
func (a *ElementActions) OnMouseLeave(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mouseleave", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// OnMouseOver registers a handler for mouseover events.
func (a *ElementActions) OnMouseOver(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mouseover", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// OnMouseOut registers a handler for mouseout events.
func (a *ElementActions) OnMouseOut(handler func(MouseEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("mouseout", work.Handler{
		EventOptions: metadata.EventOptions{Props: MouseEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return a
}

// ============================================================================
// Focus Events
// ============================================================================

// OnFocus registers a handler for focus events.
func (a *ElementActions) OnFocus(handler func(FocusEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("focus", work.Handler{
		EventOptions: metadata.EventOptions{Props: FocusEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildFocusEvent(evt)) },
	})
	return a
}

// OnBlur registers a handler for blur events.
func (a *ElementActions) OnBlur(handler func(FocusEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("blur", work.Handler{
		EventOptions: metadata.EventOptions{Props: FocusEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildFocusEvent(evt)) },
	})
	return a
}

// ============================================================================
// Keyboard Events
// ============================================================================

// OnKeyDown registers a handler for keydown events.
func (a *ElementActions) OnKeyDown(handler func(KeyboardEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("keydown", work.Handler{
		EventOptions: metadata.EventOptions{Props: KeyboardEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildKeyboardEvent(evt)) },
	})
	return a
}

// OnKeyUp registers a handler for keyup events.
func (a *ElementActions) OnKeyUp(handler func(KeyboardEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("keyup", work.Handler{
		EventOptions: metadata.EventOptions{Props: KeyboardEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildKeyboardEvent(evt)) },
	})
	return a
}

// OnKeyPress registers a handler for keypress events.
func (a *ElementActions) OnKeyPress(handler func(KeyboardEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("keypress", work.Handler{
		EventOptions: metadata.EventOptions{Props: KeyboardEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildKeyboardEvent(evt)) },
	})
	return a
}

// ============================================================================
// Pointer Events
// ============================================================================

// OnPointerDown registers a handler for pointerdown events.
func (a *ElementActions) OnPointerDown(handler func(PointerEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("pointerdown", work.Handler{
		EventOptions: metadata.EventOptions{Props: PointerEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return a
}

// OnPointerUp registers a handler for pointerup events.
func (a *ElementActions) OnPointerUp(handler func(PointerEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("pointerup", work.Handler{
		EventOptions: metadata.EventOptions{Props: PointerEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return a
}

// OnPointerMove registers a handler for pointermove events.
func (a *ElementActions) OnPointerMove(handler func(PointerEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("pointermove", work.Handler{
		EventOptions: metadata.EventOptions{Props: PointerEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return a
}

// OnPointerEnter registers a handler for pointerenter events.
func (a *ElementActions) OnPointerEnter(handler func(PointerEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("pointerenter", work.Handler{
		EventOptions: metadata.EventOptions{Props: PointerEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return a
}

// OnPointerLeave registers a handler for pointerleave events.
func (a *ElementActions) OnPointerLeave(handler func(PointerEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("pointerleave", work.Handler{
		EventOptions: metadata.EventOptions{Props: PointerEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return a
}

// OnPointerCancel registers a handler for pointercancel events.
func (a *ElementActions) OnPointerCancel(handler func(PointerEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("pointercancel", work.Handler{
		EventOptions: metadata.EventOptions{Props: PointerEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return a
}

// ============================================================================
// Touch Events
// ============================================================================

// OnTouchStart registers a handler for touchstart events.
func (a *ElementActions) OnTouchStart(handler func(TouchEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("touchstart", work.Handler{
		EventOptions: metadata.EventOptions{Props: TouchEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return a
}

// OnTouchEnd registers a handler for touchend events.
func (a *ElementActions) OnTouchEnd(handler func(TouchEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("touchend", work.Handler{
		EventOptions: metadata.EventOptions{Props: TouchEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return a
}

// OnTouchMove registers a handler for touchmove events.
func (a *ElementActions) OnTouchMove(handler func(TouchEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("touchmove", work.Handler{
		EventOptions: metadata.EventOptions{Props: TouchEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return a
}

// OnTouchCancel registers a handler for touchcancel events.
func (a *ElementActions) OnTouchCancel(handler func(TouchEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("touchcancel", work.Handler{
		EventOptions: metadata.EventOptions{Props: TouchEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return a
}

// ============================================================================
// Drag Events
// ============================================================================

// OnDrag registers a handler for drag events.
func (a *ElementActions) OnDrag(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("drag", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// OnDragStart registers a handler for dragstart events.
func (a *ElementActions) OnDragStart(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("dragstart", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// OnDragEnd registers a handler for dragend events.
func (a *ElementActions) OnDragEnd(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("dragend", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// OnDragEnter registers a handler for dragenter events.
func (a *ElementActions) OnDragEnter(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("dragenter", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// OnDragLeave registers a handler for dragleave events.
func (a *ElementActions) OnDragLeave(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("dragleave", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// OnDragOver registers a handler for dragover events.
func (a *ElementActions) OnDragOver(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("dragover", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// OnDrop registers a handler for drop events.
func (a *ElementActions) OnDrop(handler func(DragEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("drop", work.Handler{
		EventOptions: metadata.EventOptions{Props: DragEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return a
}

// ============================================================================
// Wheel Events
// ============================================================================

// OnWheel registers a handler for wheel events.
func (a *ElementActions) OnWheel(handler func(WheelEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("wheel", work.Handler{
		EventOptions: metadata.EventOptions{Props: WheelEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildWheelEvent(evt)) },
	})
	return a
}

// ============================================================================
// Animation & Transition Events
// ============================================================================

// OnAnimationStart registers a handler for animationstart events.
func (a *ElementActions) OnAnimationStart(handler func(AnimationEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("animationstart", work.Handler{
		EventOptions: metadata.EventOptions{Props: AnimationEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildAnimationEvent(evt)) },
	})
	return a
}

// OnAnimationEnd registers a handler for animationend events.
func (a *ElementActions) OnAnimationEnd(handler func(AnimationEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("animationend", work.Handler{
		EventOptions: metadata.EventOptions{Props: AnimationEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildAnimationEvent(evt)) },
	})
	return a
}

// OnAnimationIteration registers a handler for animationiteration events.
func (a *ElementActions) OnAnimationIteration(handler func(AnimationEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("animationiteration", work.Handler{
		EventOptions: metadata.EventOptions{Props: AnimationEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildAnimationEvent(evt)) },
	})
	return a
}

// OnAnimationCancel registers a handler for animationcancel events.
func (a *ElementActions) OnAnimationCancel(handler func(AnimationEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("animationcancel", work.Handler{
		EventOptions: metadata.EventOptions{Props: AnimationEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildAnimationEvent(evt)) },
	})
	return a
}

// OnTransitionStart registers a handler for transitionstart events.
func (a *ElementActions) OnTransitionStart(handler func(TransitionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("transitionstart", work.Handler{
		EventOptions: metadata.EventOptions{Props: TransitionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTransitionEvent(evt)) },
	})
	return a
}

// OnTransitionEnd registers a handler for transitionend events.
func (a *ElementActions) OnTransitionEnd(handler func(TransitionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("transitionend", work.Handler{
		EventOptions: metadata.EventOptions{Props: TransitionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTransitionEvent(evt)) },
	})
	return a
}

// OnTransitionRun registers a handler for transitionrun events.
func (a *ElementActions) OnTransitionRun(handler func(TransitionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("transitionrun", work.Handler{
		EventOptions: metadata.EventOptions{Props: TransitionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTransitionEvent(evt)) },
	})
	return a
}

// OnTransitionCancel registers a handler for transitioncancel events.
func (a *ElementActions) OnTransitionCancel(handler func(TransitionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("transitioncancel", work.Handler{
		EventOptions: metadata.EventOptions{Props: TransitionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildTransitionEvent(evt)) },
	})
	return a
}

// ============================================================================
// Clipboard Events
// ============================================================================

// OnCopy registers a handler for copy events.
func (a *ElementActions) OnCopy(handler func(ClipboardEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("copy", work.Handler{
		EventOptions: metadata.EventOptions{Props: ClipboardEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildClipboardEvent(evt)) },
	})
	return a
}

// OnCut registers a handler for cut events.
func (a *ElementActions) OnCut(handler func(ClipboardEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("cut", work.Handler{
		EventOptions: metadata.EventOptions{Props: ClipboardEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildClipboardEvent(evt)) },
	})
	return a
}

// OnPaste registers a handler for paste events.
func (a *ElementActions) OnPaste(handler func(ClipboardEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("paste", work.Handler{
		EventOptions: metadata.EventOptions{Props: ClipboardEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildClipboardEvent(evt)) },
	})
	return a
}

// ============================================================================
// Composition Events
// ============================================================================

// OnCompositionStart registers a handler for compositionstart events.
func (a *ElementActions) OnCompositionStart(handler func(CompositionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("compositionstart", work.Handler{
		EventOptions: metadata.EventOptions{Props: CompositionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildCompositionEvent(evt)) },
	})
	return a
}

// OnCompositionUpdate registers a handler for compositionupdate events.
func (a *ElementActions) OnCompositionUpdate(handler func(CompositionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("compositionupdate", work.Handler{
		EventOptions: metadata.EventOptions{Props: CompositionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildCompositionEvent(evt)) },
	})
	return a
}

// OnCompositionEnd registers a handler for compositionend events.
func (a *ElementActions) OnCompositionEnd(handler func(CompositionEvent) work.Updates) *ElementActions {
	if handler == nil {
		return a
	}
	a.addHandler("compositionend", work.Handler{
		EventOptions: metadata.EventOptions{Props: CompositionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildCompositionEvent(evt)) },
	})
	return a
}
