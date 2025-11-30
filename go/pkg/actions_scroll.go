package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type ScrollOptions struct {
	Behavior string
	Block    string
	Inline   string
}

type ScrollActions struct {
	*ElementActions
}

func NewScrollActions(ctx *runtime.Ctx, ref work.Attachment) *ScrollActions {
	return &ScrollActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *ScrollActions) ScrollIntoView(opts *ScrollOptions) {
	if opts == nil {
		a.Call("scrollIntoView")
		return
	}
	a.Call("scrollIntoView", map[string]any{
		"behavior": opts.Behavior,
		"block":    opts.Block,
		"inline":   opts.Inline,
	})
}

func (a *ScrollActions) SetScrollTop(value float64) {
	a.Set("scrollTop", value)
}

func (a *ScrollActions) SetScrollLeft(value float64) {
	a.Set("scrollLeft", value)
}

func (a *ScrollActions) GetScrollTop() (float64, error) {
	values, err := a.Query("scrollTop")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["scrollTop"]), nil
}

func (a *ScrollActions) GetScrollLeft() (float64, error) {
	values, err := a.Query("scrollLeft")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["scrollLeft"]), nil
}

func (a *ScrollActions) OnScroll(handler func(ScrollEvent) work.Updates) *ScrollActions {
	if handler == nil {
		return a
	}
	a.addHandler("scroll", work.Handler{
		EventOptions: metadata.EventOptions{Props: ScrollEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildScrollEvent(evt)) },
	})
	return a
}
