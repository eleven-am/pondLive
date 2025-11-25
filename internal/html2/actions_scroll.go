package html2

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ScrollOptions configures scroll behavior.
type ScrollOptions struct {
	Behavior string // "auto", "smooth", "instant"
	Block    string // "start", "center", "end", "nearest"
	Inline   string // "start", "center", "end", "nearest"
}

// ScrollActions provides actions and queries for scrollable elements.
//
// Example:
//
//	divRef := ui.UseRef(ctx)
//	actions := html.Scroll(ctx, divRef)
//	actions.ScrollIntoView(&html.ScrollOptions{Behavior: "smooth", Block: "center"})
//
//	return html.El("div", html.Attach(divRef), html.Text("Scroll to me"))
type ScrollActions struct {
	*ElementActions
}

// NewScrollActions creates a ScrollActions for the given ref.
func NewScrollActions(ctx *runtime2.Ctx, ref work.Attachment) *ScrollActions {
	return &ScrollActions{ElementActions: NewElementActions(ctx, ref)}
}

// ScrollIntoView scrolls the element into view with the provided options.
//
// Example:
//
//	actions := html.Scroll(ctx, ref)
//	actions.ScrollIntoView(&html.ScrollOptions{Behavior: "smooth", Block: "center"})
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

// SetScrollTop sets the vertical scroll position of the element.
//
// Example:
//
//	actions := html.Scroll(ctx, ref)
//	actions.SetScrollTop(100.0)  // Scroll down 100 pixels
func (a *ScrollActions) SetScrollTop(value float64) {
	a.Set("scrollTop", value)
}

// SetScrollLeft sets the horizontal scroll position of the element.
//
// Example:
//
//	actions := html.Scroll(ctx, ref)
//	actions.SetScrollLeft(50.0)  // Scroll right 50 pixels
func (a *ScrollActions) SetScrollLeft(value float64) {
	a.Set("scrollLeft", value)
}

// GetScrollTop retrieves the current vertical scroll position.
func (a *ScrollActions) GetScrollTop() (float64, error) {
	values, err := a.Query("scrollTop")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["scrollTop"]), nil
}

// GetScrollLeft retrieves the current horizontal scroll position.
func (a *ScrollActions) GetScrollLeft() (float64, error) {
	values, err := a.Query("scrollLeft")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["scrollLeft"]), nil
}

// OnScroll attaches a scroll event handler.
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
