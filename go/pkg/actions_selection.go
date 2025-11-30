package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type SelectionActions struct {
	*ValueActions
}

func NewSelectionActions(ctx *runtime.Ctx, ref work.Attachment) *SelectionActions {
	return &SelectionActions{ValueActions: NewValueActions(ctx, ref)}
}

func (a *SelectionActions) Select() {
	a.Call("select")
}

func (a *SelectionActions) SetSelectionRange(start, end int, direction ...string) {
	if len(direction) > 0 {
		a.Call("setSelectionRange", start, end, direction[0])
	} else {
		a.Call("setSelectionRange", start, end)
	}
}

func (a *SelectionActions) SetRangeText(text string, startEnd ...int) {
	if len(startEnd) >= 2 {
		a.Call("setRangeText", text, startEnd[0], startEnd[1])
	} else {
		a.Call("setRangeText", text)
	}
}

func (a *SelectionActions) GetSelectionStart() (int, error) {
	values, err := a.Query("selectionStart")
	if err != nil {
		return 0, err
	}
	return toInt(values["selectionStart"]), nil
}

func (a *SelectionActions) GetSelectionEnd() (int, error) {
	values, err := a.Query("selectionEnd")
	if err != nil {
		return 0, err
	}
	return toInt(values["selectionEnd"]), nil
}

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
