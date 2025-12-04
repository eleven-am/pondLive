package pkg

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type FormActions struct {
	*ElementActions
}

func NewFormActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *FormActions {
	return &FormActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *FormActions) Submit() {
	a.Call("submit")
}

func (a *FormActions) Reset() {
	a.Call("reset")
}

func (a *FormActions) RequestSubmit() {
	a.Call("requestSubmit")
}

func (a *FormActions) CheckValidity() (bool, error) {
	result, err := a.AsyncCall("checkValidity")
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

func (a *FormActions) ReportValidity() (bool, error) {
	result, err := a.AsyncCall("reportValidity")
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, nil
}

func (a *FormActions) OnSubmit(handler func(FormEvent) work.Updates, opts ...metadata.EventOptions) *FormActions {
	if handler == nil {
		return a
	}
	a.addHandler("submit", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: FormEvent{}.props(), Prevent: true}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildFormEvent(evt)) },
	})
	return a
}

func (a *FormActions) OnReset(handler func(FormEvent) work.Updates, opts ...metadata.EventOptions) *FormActions {
	if handler == nil {
		return a
	}
	a.addHandler("reset", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: FormEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildFormEvent(evt)) },
	})
	return a
}
