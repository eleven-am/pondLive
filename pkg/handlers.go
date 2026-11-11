package pkg

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/work"
)

func OnInvalid(h func(Event), opts ...EventOpt) Item {
	if h == nil {
		return work.Skip()
	}
	return OnWith("invalid", applyOpts(EventOptions{}, opts), h)
}

func OnClick(h func(ClickEvent), opts ...EventOpt) Item {
	return typedOn[ClickEvent]("click", EventOptions{}, h, opts)
}

func OnDoubleClick(h func(ClickEvent), opts ...EventOpt) Item {
	return typedOn[ClickEvent]("dblclick", EventOptions{}, h, opts)
}

func OnContextMenu(h func(ClickEvent), opts ...EventOpt) Item {
	return typedOn[ClickEvent]("contextmenu", EventOptions{}, h, opts)
}

func OnAuxClick(h func(ClickEvent), opts ...EventOpt) Item {
	return typedOn[ClickEvent]("auxclick", EventOptions{}, h, opts)
}

func OnMouseDown(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mousedown", EventOptions{}, h, opts)
}

func OnMouseUp(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mouseup", EventOptions{}, h, opts)
}

func OnMouseMove(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mousemove", EventOptions{}, h, opts)
}

func OnMouseEnter(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mouseenter", EventOptions{}, h, opts)
}

func OnMouseLeave(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mouseleave", EventOptions{}, h, opts)
}

func OnMouseOver(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mouseover", EventOptions{}, h, opts)
}

func OnMouseOut(h func(MouseEvent), opts ...EventOpt) Item {
	return typedOn[MouseEvent]("mouseout", EventOptions{}, h, opts)
}

func OnFocus(h func(FocusEvent), opts ...EventOpt) Item {
	return typedOn[FocusEvent]("focus", EventOptions{}, h, opts)
}

func OnBlur(h func(FocusEvent), opts ...EventOpt) Item {
	return typedOn[FocusEvent]("blur", EventOptions{}, h, opts)
}

func OnFocusIn(h func(FocusEvent), opts ...EventOpt) Item {
	return typedOn[FocusEvent]("focusin", EventOptions{}, h, opts)
}

func OnFocusOut(h func(FocusEvent), opts ...EventOpt) Item {
	return typedOn[FocusEvent]("focusout", EventOptions{}, h, opts)
}

func OnKeyDown(h func(KeyboardEvent), opts ...EventOpt) Item {
	return typedOn[KeyboardEvent]("keydown", EventOptions{}, h, opts)
}

func OnKeyUp(h func(KeyboardEvent), opts ...EventOpt) Item {
	return typedOn[KeyboardEvent]("keyup", EventOptions{}, h, opts)
}

func OnPointerDown(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointerdown", EventOptions{}, h, opts)
}

func OnPointerUp(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointerup", EventOptions{}, h, opts)
}

func OnPointerMove(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointermove", EventOptions{}, h, opts)
}

func OnPointerEnter(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointerenter", EventOptions{}, h, opts)
}

func OnPointerLeave(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointerleave", EventOptions{}, h, opts)
}

func OnPointerCancel(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointercancel", EventOptions{}, h, opts)
}

func OnPointerOver(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointerover", EventOptions{}, h, opts)
}

func OnPointerOut(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("pointerout", EventOptions{}, h, opts)
}

func OnLostPointerCapture(h func(PointerEvent), opts ...EventOpt) Item {
	return typedOn[PointerEvent]("lostpointercapture", EventOptions{}, h, opts)
}

func OnTouchStart(h func(TouchEvent), opts ...EventOpt) Item {
	return typedOn[TouchEvent]("touchstart", EventOptions{}, h, opts)
}

func OnTouchEnd(h func(TouchEvent), opts ...EventOpt) Item {
	return typedOn[TouchEvent]("touchend", EventOptions{}, h, opts)
}

func OnTouchMove(h func(TouchEvent), opts ...EventOpt) Item {
	return typedOn[TouchEvent]("touchmove", EventOptions{}, h, opts)
}

func OnTouchCancel(h func(TouchEvent), opts ...EventOpt) Item {
	return typedOn[TouchEvent]("touchcancel", EventOptions{}, h, opts)
}

func OnDrag(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("drag", EventOptions{}, h, opts)
}

func OnDragStart(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("dragstart", EventOptions{}, h, opts)
}

func OnDragEnd(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("dragend", EventOptions{}, h, opts)
}

func OnDragEnter(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("dragenter", EventOptions{}, h, opts)
}

func OnDragLeave(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("dragleave", EventOptions{}, h, opts)
}

func OnDragOver(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("dragover", EventOptions{}, h, opts)
}

func OnDrop(h func(DragEvent), opts ...EventOpt) Item {
	return typedOn[DragEvent]("drop", EventOptions{}, h, opts)
}

func OnWheel(h func(WheelEvent), opts ...EventOpt) Item {
	return typedOn[WheelEvent]("wheel", EventOptions{}, h, opts)
}

func OnScroll(h func(ScrollEvent), opts ...EventOpt) Item {
	return typedOn[ScrollEvent]("scroll", EventOptions{}, h, opts)
}

func OnScrollEnd(h func(ScrollEvent), opts ...EventOpt) Item {
	return typedOn[ScrollEvent]("scrollend", EventOptions{}, h, opts)
}

func OnAnimationStart(h func(AnimationEvent), opts ...EventOpt) Item {
	return typedOn[AnimationEvent]("animationstart", EventOptions{}, h, opts)
}

func OnAnimationEnd(h func(AnimationEvent), opts ...EventOpt) Item {
	return typedOn[AnimationEvent]("animationend", EventOptions{}, h, opts)
}

func OnAnimationIteration(h func(AnimationEvent), opts ...EventOpt) Item {
	return typedOn[AnimationEvent]("animationiteration", EventOptions{}, h, opts)
}

func OnAnimationCancel(h func(AnimationEvent), opts ...EventOpt) Item {
	return typedOn[AnimationEvent]("animationcancel", EventOptions{}, h, opts)
}

func OnTransitionStart(h func(TransitionEvent), opts ...EventOpt) Item {
	return typedOn[TransitionEvent]("transitionstart", EventOptions{}, h, opts)
}

func OnTransitionEnd(h func(TransitionEvent), opts ...EventOpt) Item {
	return typedOn[TransitionEvent]("transitionend", EventOptions{}, h, opts)
}

func OnTransitionRun(h func(TransitionEvent), opts ...EventOpt) Item {
	return typedOn[TransitionEvent]("transitionrun", EventOptions{}, h, opts)
}

func OnTransitionCancel(h func(TransitionEvent), opts ...EventOpt) Item {
	return typedOn[TransitionEvent]("transitioncancel", EventOptions{}, h, opts)
}

func OnCopy(h func(ClipboardEvent), opts ...EventOpt) Item {
	return typedOn[ClipboardEvent]("copy", EventOptions{}, h, opts)
}

func OnCut(h func(ClipboardEvent), opts ...EventOpt) Item {
	return typedOn[ClipboardEvent]("cut", EventOptions{}, h, opts)
}

func OnPaste(h func(ClipboardEvent), opts ...EventOpt) Item {
	return typedOn[ClipboardEvent]("paste", EventOptions{}, h, opts)
}

func OnCompositionStart(h func(CompositionEvent), opts ...EventOpt) Item {
	return typedOn[CompositionEvent]("compositionstart", EventOptions{}, h, opts)
}

func OnCompositionUpdate(h func(CompositionEvent), opts ...EventOpt) Item {
	return typedOn[CompositionEvent]("compositionupdate", EventOptions{}, h, opts)
}

func OnCompositionEnd(h func(CompositionEvent), opts ...EventOpt) Item {
	return typedOn[CompositionEvent]("compositionend", EventOptions{}, h, opts)
}

func OnSubmit(h func(FormEvent), opts ...EventOpt) Item {
	return typedOn[FormEvent]("submit", EventOptions{Prevent: true}, h, opts)
}

func OnReset(h func(FormEvent), opts ...EventOpt) Item {
	return typedOn[FormEvent]("reset", EventOptions{}, h, opts)
}

func OnPlay(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("play", EventOptions{}, h, opts)
}

func OnPause(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("pause", EventOptions{}, h, opts)
}

func OnEnded(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("ended", EventOptions{}, h, opts)
}

func OnTimeUpdate(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("timeupdate", EventOptions{}, h, opts)
}

func OnVolumeChange(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("volumechange", EventOptions{}, h, opts)
}

func OnSeeking(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("seeking", EventOptions{}, h, opts)
}

func OnSeeked(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("seeked", EventOptions{}, h, opts)
}

func OnLoadedMetadata(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("loadedmetadata", EventOptions{}, h, opts)
}

func OnLoadedData(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("loadeddata", EventOptions{}, h, opts)
}

func OnCanPlay(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("canplay", EventOptions{}, h, opts)
}

func OnCanPlayThrough(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("canplaythrough", EventOptions{}, h, opts)
}

func OnWaiting(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("waiting", EventOptions{}, h, opts)
}

func OnPlaying(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("playing", EventOptions{}, h, opts)
}

func OnStalled(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("stalled", EventOptions{}, h, opts)
}

func OnRateChange(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("ratechange", EventOptions{}, h, opts)
}

func OnDurationChange(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("durationchange", EventOptions{}, h, opts)
}

func OnLoadStart(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("loadstart", EventOptions{}, h, opts)
}

func OnSuspend(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("suspend", EventOptions{}, h, opts)
}

func OnAbort(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("abort", EventOptions{}, h, opts)
}

func OnEmptied(h func(MediaEvent), opts ...EventOpt) Item {
	return typedOn[MediaEvent]("emptied", EventOptions{}, h, opts)
}

func OnProgress(h func(ProgressEvent), opts ...EventOpt) Item {
	return typedOn[ProgressEvent]("progress", EventOptions{}, h, opts)
}

func OnError(h func(ErrorEvent), opts ...EventOpt) Item {
	return typedOn[ErrorEvent]("error", EventOptions{}, h, opts)
}

func OnLoad(h func(LoadEvent), opts ...EventOpt) Item {
	return typedOn[LoadEvent]("load", EventOptions{}, h, opts)
}

func OnToggle(h func(ToggleEvent), opts ...EventOpt) Item {
	return typedOn[ToggleEvent]("toggle", EventOptions{}, h, opts)
}

func OnBeforeToggle(h func(ToggleEvent), opts ...EventOpt) Item {
	return typedOn[ToggleEvent]("beforetoggle", EventOptions{}, h, opts)
}

func OnClose(h func(DialogEvent), opts ...EventOpt) Item {
	return typedOn[DialogEvent]("close", EventOptions{}, h, opts)
}

func OnCancel(h func(DialogEvent), opts ...EventOpt) Item {
	return typedOn[DialogEvent]("cancel", EventOptions{}, h, opts)
}

func OnSelect(h func(SelectionEvent), opts ...EventOpt) Item {
	return typedOn[SelectionEvent]("select", EventOptions{}, h, opts)
}

func OnSelectionChange(h func(SelectionEvent), opts ...EventOpt) Item {
	return typedOn[SelectionEvent]("selectionchange", EventOptions{}, h, opts)
}

func OnBeforeInput(h func(BeforeInputEvent), opts ...EventOpt) Item {
	return typedOn[BeforeInputEvent]("beforeinput", EventOptions{}, h, opts)
}

func OnFullscreenChange(h func(FullscreenChangeEvent), opts ...EventOpt) Item {
	return typedOn[FullscreenChangeEvent]("fullscreenchange", EventOptions{}, h, opts)
}

func OnChange(h func(ChangeEvent), opts ...EventOpt) Item {
	return typedOn[ChangeEvent](metadata.EditEvent, EventOptions{}, h, opts)
}
