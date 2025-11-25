package html2

import (
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// SelectionActions provides text selection actions for input and textarea elements.
//
// Example:
//
//	inputRef := ui.UseRef(ctx)
//	actions := html.Selection(ctx, inputRef)
//	actions.Select()  // Select all text
//
//	return html.El("input", html.Attach(inputRef), html.Type("text"))
type SelectionActions struct {
	*ValueActions
}

// NewSelectionActions creates a SelectionActions for the given ref.
func NewSelectionActions(ctx *runtime2.Ctx, ref work.Attachment) *SelectionActions {
	return &SelectionActions{ValueActions: NewValueActions(ctx, ref)}
}

// Select selects all text in the input or textarea element.
//
// Example:
//
//	actions := html.Selection(ctx, inputRef)
//	actions.Select()
func (a *SelectionActions) Select() {
	a.Call("select")
}

// SetSelectionRange sets the start and end positions of text selection.
// Direction can be "forward", "backward", or "none".
//
// Example:
//
//	actions := html.Selection(ctx, inputRef)
//	actions.SetSelectionRange(0, 5)  // Select first 5 characters
//	actions.SetSelectionRange(0, 5, "forward")  // With direction
func (a *SelectionActions) SetSelectionRange(start, end int, direction ...string) {
	if len(direction) > 0 {
		a.Call("setSelectionRange", start, end, direction[0])
	} else {
		a.Call("setSelectionRange", start, end)
	}
}

// SetRangeText replaces a range of text with new text.
// If start and end are provided, replaces that range.
// Otherwise replaces the current selection.
//
// Example:
//
//	actions := html.Selection(ctx, inputRef)
//	actions.SetRangeText("new text")  // Replace selection
//	actions.SetRangeText("new text", 0, 5)  // Replace range 0-5
func (a *SelectionActions) SetRangeText(text string, startEnd ...int) {
	if len(startEnd) >= 2 {
		a.Call("setRangeText", text, startEnd[0], startEnd[1])
	} else {
		a.Call("setRangeText", text)
	}
}

// GetSelectionStart retrieves the start position of the text selection.
func (a *SelectionActions) GetSelectionStart() (int, error) {
	values, err := a.Query("selectionStart")
	if err != nil {
		return 0, err
	}
	return toInt(values["selectionStart"]), nil
}

// GetSelectionEnd retrieves the end position of the text selection.
func (a *SelectionActions) GetSelectionEnd() (int, error) {
	values, err := a.Query("selectionEnd")
	if err != nil {
		return 0, err
	}
	return toInt(values["selectionEnd"]), nil
}

// GetSelectionDirection retrieves the direction of the text selection.
// Returns "forward", "backward", or "none".
func (a *SelectionActions) GetSelectionDirection() (string, error) {
	values, err := a.Query("selectionDirection")
	if err != nil {
		return "", err
	}
	if s, ok := values["selectionDirection"].(string); ok {
		return s, nil
	}
	return "", nil
}
