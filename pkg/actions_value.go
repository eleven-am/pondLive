package pkg

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type ValueActions struct {
	*ElementActions
}

func NewValueActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *ValueActions {
	return &ValueActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *ValueActions) SetValue(value string) {
	a.Set("value", value)
}

func (a *ValueActions) SetChecked(checked bool) {
	a.Set("checked", checked)
}

func (a *ValueActions) SetSelectedIndex(index int) {
	a.Set("selectedIndex", index)
}

func (a *ValueActions) GetValue() (string, error) {
	values, err := a.Query("value")
	if err != nil {
		return "", err
	}
	if s, ok := values["value"].(string); ok {
		return s, nil
	}
	return "", nil
}

func (a *ValueActions) GetChecked() (bool, error) {
	values, err := a.Query("checked")
	if err != nil {
		return false, err
	}
	if b, ok := values["checked"].(bool); ok {
		return b, nil
	}
	return false, nil
}

func (a *ValueActions) GetSelectedIndex() (int, error) {
	values, err := a.Query("selectedIndex")
	if err != nil {
		return -1, err
	}
	return toInt(values["selectedIndex"]), nil
}

func (a *ValueActions) OnInput(handler func(InputEvent) work.Updates, opts ...metadata.EventOptions) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("input", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: InputEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildInputEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnChange(handler func(ChangeEvent) work.Updates, opts ...metadata.EventOptions) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("input", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ChangeEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildChangeEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnFileChange(handler func(ChangeEvent) work.Updates, opts ...metadata.EventOptions) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("change", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ChangeEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildChangeEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnSelect(handler func(SelectionEvent) work.Updates, opts ...metadata.EventOptions) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("select", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: SelectionEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildSelectionEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnInvalid(handler func(work.Event) work.Updates, opts ...metadata.EventOptions) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("invalid", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{}, opts...),
		Fn:           handler,
	})
	return a
}
