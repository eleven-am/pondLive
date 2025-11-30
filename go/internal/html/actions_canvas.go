package html

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type CanvasActions struct {
	*ElementActions
}

func NewCanvasActions(ctx *runtime.Ctx, ref work.Attachment) *CanvasActions {
	return &CanvasActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *CanvasActions) ToDataURL(mimeType ...string) (string, error) {
	var result any
	var err error
	if len(mimeType) > 0 {
		result, err = a.AsyncCall("toDataURL", mimeType[0])
	} else {
		result, err = a.AsyncCall("toDataURL")
	}
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

func (a *CanvasActions) ToDataURLWithQuality(mimeType string, quality float64) (string, error) {
	result, err := a.AsyncCall("toDataURL", mimeType, quality)
	if err != nil {
		return "", err
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return "", nil
}

func (a *CanvasActions) GetWidth() (int, error) {
	values, err := a.Query("width")
	if err != nil {
		return 0, err
	}
	return toInt(values["width"]), nil
}

func (a *CanvasActions) GetHeight() (int, error) {
	values, err := a.Query("height")
	if err != nil {
		return 0, err
	}
	return toInt(values["height"]), nil
}

func (a *CanvasActions) SetWidth(width int) {
	a.Set("width", width)
}

func (a *CanvasActions) SetHeight(height int) {
	a.Set("height", height)
}
