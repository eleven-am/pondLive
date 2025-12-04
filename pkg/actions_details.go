package pkg

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type DetailsActions struct {
	*ElementActions
}

func NewDetailsActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *DetailsActions {
	return &DetailsActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *DetailsActions) SetOpen(open bool) {
	a.Set("open", open)
}

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

func (a *DetailsActions) Toggle() error {
	open, err := a.GetOpen()
	if err != nil {
		return err
	}
	a.SetOpen(!open)
	return nil
}

func (a *DetailsActions) OnToggle(handler func(ToggleEvent) work.Updates, opts ...metadata.EventOptions) *DetailsActions {
	if handler == nil {
		return a
	}
	a.addHandler("toggle", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ToggleEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildToggleEvent(evt)) },
	})
	return a
}
