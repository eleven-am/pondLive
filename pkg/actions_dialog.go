package pkg

import (
	"github.com/eleven-am/pondlive/internal/runtime"
)

type DialogActions struct {
	*ElementActions
}

func NewDialogActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *DialogActions {
	return &DialogActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *DialogActions) Show() {
	a.Call("show")
}

func (a *DialogActions) ShowModal() {
	a.Call("showModal")
}

func (a *DialogActions) Close(returnValue ...string) {
	if len(returnValue) > 0 {
		a.Call("close", returnValue[0])
	} else {
		a.Call("close")
	}
}

func (a *DialogActions) GetOpen() (bool, error) {
	values, err := a.Query("open")
	if err != nil {
		return false, err
	}
	if b, ok := values["open"].(bool); ok {
		return b, nil
	}
	return false, nil
}

func (a *DialogActions) GetReturnValue() (string, error) {
	values, err := a.Query("returnValue")
	if err != nil {
		return "", err
	}
	if s, ok := values["returnValue"].(string); ok {
		return s, nil
	}
	return "", nil
}
