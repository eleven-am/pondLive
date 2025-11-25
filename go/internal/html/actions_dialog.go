package html

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// DialogActions provides dialog-related DOM actions.
//
// Example:
//
//	dialogRef := ui.UseRef(ctx)
//	actions := html.Dialog(ctx, dialogRef)
//	actions.ShowModal()
//
//	return html.El("dialog", html.Attach(dialogRef), ...)
type DialogActions struct {
	*ElementActions
}

// NewDialogActions creates a DialogActions for the given ref.
func NewDialogActions(ctx *runtime.Ctx, ref work.Attachment) *DialogActions {
	return &DialogActions{ElementActions: NewElementActions(ctx, ref)}
}

// Show opens the dialog as non-modal.
// The dialog appears but the user can still interact with the rest of the page.
//
// Example:
//
//	actions := html.Dialog(ctx, dialogRef)
//	actions.Show()
func (a *DialogActions) Show() {
	a.Call("show")
}

// ShowModal opens the dialog as modal.
// The dialog appears with a backdrop and the user cannot interact with the rest of the page.
//
// Example:
//
//	actions := html.Dialog(ctx, dialogRef)
//	actions.ShowModal()
func (a *DialogActions) ShowModal() {
	a.Call("showModal")
}

// Close closes the dialog with an optional return value.
//
// Example:
//
//	actions := html.Dialog(ctx, dialogRef)
//	actions.Close()  // Close without return value
//	actions.Close("confirmed")  // Close with return value
func (a *DialogActions) Close(returnValue ...string) {
	if len(returnValue) > 0 {
		a.Call("close", returnValue[0])
	} else {
		a.Call("close")
	}
}

// GetOpen retrieves whether the dialog is currently open.
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

// GetReturnValue retrieves the dialog's return value (set by Close).
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
