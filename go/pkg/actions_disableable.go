package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type DisableableActions struct {
	*ElementActions
}

func NewDisableableActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *DisableableActions {
	return &DisableableActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *DisableableActions) SetDisabled(disabled bool) {
	a.Set("disabled", disabled)
}

func (a *DisableableActions) SetEnabled(enabled bool) {
	a.Set("disabled", !enabled)
}

func (a *DisableableActions) GetDisabled() (bool, error) {
	values, err := a.Query("disabled")
	if err != nil {
		return false, err
	}
	if b, ok := values["disabled"].(bool); ok {
		return b, nil
	}
	return false, nil
}
