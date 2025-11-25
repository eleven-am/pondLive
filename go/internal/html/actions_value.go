package html

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ValueActions provides value-related DOM property setters for form elements.
// Embedded in refs for input, textarea, select, and other form controls.
//
// Example:
//
//	inputRef := ui.UseRef(ctx)
//	actions := html.Input(ctx, inputRef)
//	actions.SetValue("New value")
//
//	return html.El("input", html.Attach(inputRef), html.Type("text"))
type ValueActions struct {
	*ElementActions
}

// Input creates a ValueActions for the given ref.
func NewValueActions(ctx *runtime.Ctx, ref work.Attachment) *ValueActions {
	return &ValueActions{ElementActions: NewElementActions(ctx, ref)}
}

// SetValue sets the value property of the element.
//
// Example:
//
//	actions := html.Input(ctx, inputRef)
//	actions.SetValue("New value")
func (a *ValueActions) SetValue(value string) {
	a.Set("value", value)
}

// SetChecked sets the checked property of the element (for checkboxes and radio buttons).
//
// Example:
//
//	actions := html.Input(ctx, checkboxRef)
//	actions.SetChecked(true)
func (a *ValueActions) SetChecked(checked bool) {
	a.Set("checked", checked)
}

// SetSelectedIndex sets the selectedIndex property of a select element.
//
// Example:
//
//	actions := html.Input(ctx, selectRef)
//	actions.SetSelectedIndex(2)  // Select the third option (0-indexed)
func (a *ValueActions) SetSelectedIndex(index int) {
	a.Set("selectedIndex", index)
}

// GetValue retrieves the current value of the element.
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

// GetChecked retrieves the checked state of the element.
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

// GetSelectedIndex retrieves the selected index of a select element.
func (a *ValueActions) GetSelectedIndex() (int, error) {
	values, err := a.Query("selectedIndex")
	if err != nil {
		return -1, err
	}
	return toInt(values["selectedIndex"]), nil
}

func (a *ValueActions) OnInput(handler func(InputEvent) work.Updates) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("input", work.Handler{
		EventOptions: metadata.EventOptions{Props: InputEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildInputEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnChange(handler func(ChangeEvent) work.Updates) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("change", work.Handler{
		EventOptions: metadata.EventOptions{Props: ChangeEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildChangeEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnSelect(handler func(SelectionEvent) work.Updates) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("select", work.Handler{
		EventOptions: metadata.EventOptions{Props: SelectionEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildSelectionEvent(evt)) },
	})
	return a
}

func (a *ValueActions) OnInvalid(handler func(work.Event) work.Updates) *ValueActions {
	if handler == nil {
		return a
	}
	a.addHandler("invalid", work.Handler{
		Fn: handler,
	})
	return a
}
