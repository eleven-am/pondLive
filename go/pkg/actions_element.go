package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type DOMRect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

type ScrollMetrics struct {
	ScrollTop    float64
	ScrollLeft   float64
	ScrollHeight float64
	ScrollWidth  float64
	ClientHeight float64
	ClientWidth  float64
}

type ElementActions struct {
	ctx *runtime.Ctx
	ref *runtime.ElementRef
}

func NewElementActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *ElementActions {
	return &ElementActions{ctx: ctx, ref: ref}
}

func (a *ElementActions) Call(method string, args ...any) {
	if a.ctx == nil || a.ref == nil {
		return
	}
	a.ctx.Call(a.ref, method, args...)
}

func (a *ElementActions) Set(prop string, value any) {
	if a.ctx == nil || a.ref == nil {
		return
	}
	a.ctx.Set(a.ref, prop, value)
}

func (a *ElementActions) Query(selectors ...string) (map[string]any, error) {
	if a.ctx == nil || a.ref == nil {
		return nil, runtime.ErrNilRef
	}
	return a.ctx.Query(a.ref, selectors...)
}

func (a *ElementActions) AsyncCall(method string, args ...any) (any, error) {
	if a.ctx == nil || a.ref == nil {
		return nil, runtime.ErrNilRef
	}
	return a.ctx.AsyncCall(a.ref, method, args...)
}

func (a *ElementActions) Focus() {
	a.Call("focus")
}

func (a *ElementActions) Blur() {
	a.Call("blur")
}

func (a *ElementActions) Click() {
	a.Call("click")
}

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

func (a *ElementActions) addHandler(event string, handler work.Handler) {
	if a.ref == nil {
		return
	}
	a.ref.AddHandler(event, handler)
}

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
