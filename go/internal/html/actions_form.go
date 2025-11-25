package html

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// FormActions provides form-related DOM actions.
//
// Example:
//
//	formRef := ui.UseRef(ctx)
//	actions := html.Form(ctx, formRef)
//	actions.Submit()
//
//	return html.El("form", html.Attach(formRef), ...)
type FormActions struct {
	*ElementActions
}

// Form creates a FormActions for the given ref.
func NewFormActions(ctx *runtime.Ctx, ref work.Attachment) *FormActions {
	return &FormActions{ElementActions: NewElementActions(ctx, ref)}
}

// Submit submits the form programmatically.
// This bypasses form validation - use RequestSubmit for validation.
//
// Example:
//
//	actions := html.Form(ctx, formRef)
//	actions.Submit()
func (a *FormActions) Submit() {
	a.Call("submit")
}

// Reset resets the form to its initial values.
//
// Example:
//
//	actions := html.Form(ctx, formRef)
//	actions.Reset()
func (a *FormActions) Reset() {
	a.Call("reset")
}

// RequestSubmit triggers form validation and submits if valid.
// Unlike Submit(), this respects form validation and fires submit event.
//
// Example:
//
//	actions := html.Form(ctx, formRef)
//	actions.RequestSubmit()
func (a *FormActions) RequestSubmit() {
	a.Call("requestSubmit")
}

// CheckValidity checks if the form is valid without showing validation UI.
// Returns true if all form controls satisfy their constraints.
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

// ReportValidity checks validity and shows validation messages to the user.
// Returns true if valid, false if invalid (and shows UI feedback).
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

func (a *FormActions) OnSubmit(handler func(FormEvent) work.Updates) *FormActions {
	if handler == nil {
		return a
	}
	a.addHandler("submit", work.Handler{
		EventOptions: metadata.EventOptions{Props: FormEvent{}.props(), Prevent: true},
		Fn:           func(evt work.Event) work.Updates { return handler(buildFormEvent(evt)) },
	})
	return a
}

func (a *FormActions) OnReset(handler func(FormEvent) work.Updates) *FormActions {
	if handler == nil {
		return a
	}
	a.addHandler("reset", work.Handler{
		EventOptions: metadata.EventOptions{Props: FormEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildFormEvent(evt)) },
	})
	return a
}
