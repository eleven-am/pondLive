package pkg

import (
	"github.com/eleven-am/pondlive/internal/document"
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/work"
)

type DocumentActions struct {
	handle *document.DocumentHandle
}

func newDocumentActions(handle *document.DocumentHandle) *DocumentActions {
	return &DocumentActions{handle: handle}
}

func (d *DocumentActions) Html(items ...Item) *DocumentActions {
	if d.handle == nil {
		return d
	}
	workItems := make([]work.Item, len(items))
	for i, item := range items {
		workItems[i] = item
	}
	d.handle.Html(workItems...)
	return d
}

func (d *DocumentActions) Body(items ...Item) *DocumentActions {
	if d.handle == nil {
		return d
	}
	workItems := make([]work.Item, len(items))
	for i, item := range items {
		workItems[i] = item
	}
	d.handle.Body(workItems...)
	return d
}

func (d *DocumentActions) addHandler(event string, handler work.Handler) {
	if d.handle == nil {
		return
	}
	d.handle.AddBodyHandler(event, handler)
}

func (d *DocumentActions) OnClick(handler func(ClickEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("click", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ClickEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildClickEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDoubleClick(handler func(ClickEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("dblclick", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ClickEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildClickEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnContextMenu(handler func(ClickEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("contextmenu", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ClickEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildClickEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnMouseDown(handler func(MouseEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("mousedown", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MouseEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnMouseUp(handler func(MouseEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("mouseup", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MouseEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnMouseMove(handler func(MouseEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("mousemove", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MouseEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMouseEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnKeyDown(handler func(KeyboardEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("keydown", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: KeyboardEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildKeyboardEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnKeyUp(handler func(KeyboardEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("keyup", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: KeyboardEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildKeyboardEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnKeyPress(handler func(KeyboardEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("keypress", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: KeyboardEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildKeyboardEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnFocus(handler func(FocusEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("focus", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: FocusEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildFocusEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnBlur(handler func(FocusEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("blur", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: FocusEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildFocusEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnScroll(handler func(ScrollEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("scroll", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ScrollEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildScrollEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnWheel(handler func(WheelEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("wheel", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: WheelEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildWheelEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnPointerDown(handler func(PointerEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("pointerdown", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: PointerEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnPointerUp(handler func(PointerEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("pointerup", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: PointerEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnPointerMove(handler func(PointerEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("pointermove", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: PointerEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildPointerEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnTouchStart(handler func(TouchEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("touchstart", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: TouchEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnTouchEnd(handler func(TouchEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("touchend", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: TouchEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnTouchMove(handler func(TouchEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("touchmove", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: TouchEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildTouchEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnCopy(handler func(ClipboardEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("copy", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ClipboardEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildClipboardEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnCut(handler func(ClipboardEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("cut", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ClipboardEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildClipboardEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnPaste(handler func(ClipboardEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("paste", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ClipboardEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildClipboardEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDragStart(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("dragstart", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDrag(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("drag", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDragEnd(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("dragend", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDragEnter(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("dragenter", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDragLeave(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("dragleave", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDragOver(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("dragover", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnDrop(handler func(DragEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("drop", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: DragEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildDragEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnFullscreenChange(handler func(FullscreenEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("fullscreenchange", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: FullscreenEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildFullscreenEvent(evt)) },
	})
	return d
}

func (d *DocumentActions) OnVisibilityChange(handler func(BaseEvent) work.Updates, opts ...metadata.EventOptions) *DocumentActions {
	if handler == nil {
		return d
	}
	d.addHandler("visibilitychange", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(BaseEvent{Event: evt}) },
	})
	return d
}
