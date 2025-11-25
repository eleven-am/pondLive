package html

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// DisableableActions provides actions for elements that can be disabled.
// This includes buttons and form controls (input, select, textarea, fieldset, etc).
//
// Example:
//
//	buttonRef := ui.UseRef(ctx)
//	actions := html.Disableable(ctx, buttonRef)
//	actions.SetDisabled(true)  // Disable the button
//
//	return html.El("button", html.Attach(buttonRef), html.Text("Submit"))
type DisableableActions struct {
	*ElementActions
}

// NewDisableableActions creates a DisableableActions for the given ref.
func NewDisableableActions(ctx *runtime.Ctx, ref work.Attachment) *DisableableActions {
	return &DisableableActions{ElementActions: NewElementActions(ctx, ref)}
}

// SetDisabled sets the disabled property of the element.
//
// Example:
//
//	actions := html.Disableable(ctx, buttonRef)
//	actions.SetDisabled(true)  // Disable the button
func (a *DisableableActions) SetDisabled(disabled bool) {
	a.Set("disabled", disabled)
}

// SetEnabled is a convenience method that sets disabled to false.
//
// Example:
//
//	actions := html.Disableable(ctx, inputRef)
//	actions.SetEnabled(true)  // Enable the input field
func (a *DisableableActions) SetEnabled(enabled bool) {
	a.Set("disabled", !enabled)
}

// GetDisabled retrieves whether the element is currently disabled.
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
