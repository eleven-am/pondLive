package html

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// DetailsActions provides details element actions.
//
// Example:
//
//	detailsRef := ui.UseRef(ctx)
//	actions := html.Details(ctx, detailsRef)
//	actions.SetOpen(true)  // Expand the details
//
//	return html.El("details", html.Attach(detailsRef),
//	    html.El("summary", html.Text("Click to expand")),
//	    html.El("p", html.Text("Hidden content")),
//	)
type DetailsActions struct {
	*ElementActions
}

// Details creates a DetailsActions for the given ref.
func NewDetailsActions(ctx *runtime.Ctx, ref work.Attachment) *DetailsActions {
	return &DetailsActions{ElementActions: NewElementActions(ctx, ref)}
}

// SetOpen sets the open state of the details element.
//
// Example:
//
//	actions := html.Details(ctx, detailsRef)
//	actions.SetOpen(true)  // Expand
//	actions.SetOpen(false)  // Collapse
func (a *DetailsActions) SetOpen(open bool) {
	a.Set("open", open)
}

// GetOpen retrieves whether the details element is currently open.
func (a *DetailsActions) GetOpen() (bool, error) {
	values, err := a.Query("open")
	if err != nil {
		return false, err
	}
	if b, ok := values["open"].(bool); ok {
		return b, nil
	}
	return false, nil
}

// Toggle toggles the open state of the details element.
func (a *DetailsActions) Toggle() error {
	open, err := a.GetOpen()
	if err != nil {
		return err
	}
	a.SetOpen(!open)
	return nil
}

func (a *DetailsActions) OnToggle(handler func(ToggleEvent) work.Updates) *DetailsActions {
	if handler == nil {
		return a
	}
	a.addHandler("toggle", work.Handler{
		EventOptions: metadata.EventOptions{Props: ToggleEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildToggleEvent(evt)) },
	})
	return a
}
