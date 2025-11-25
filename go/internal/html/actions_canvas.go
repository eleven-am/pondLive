package html

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// CanvasActions provides canvas-related DOM actions.
//
// Example:
//
//	canvasRef := ui.UseRef(ctx)
//	actions := html.Canvas(ctx, canvasRef)
//	dataURL, _ := actions.ToDataURL("image/png")
//
//	return html.El("canvas", html.Attach(canvasRef), ...)
type CanvasActions struct {
	*ElementActions
}

// NewCanvasActions creates a CanvasActions for the given ref.
func NewCanvasActions(ctx *runtime.Ctx, ref work.Attachment) *CanvasActions {
	return &CanvasActions{ElementActions: NewElementActions(ctx, ref)}
}

// ToDataURL returns a data URL containing a representation of the canvas.
// Default MIME type is "image/png". Other options include "image/jpeg", "image/webp".
//
// Example:
//
//	actions := html.Canvas(ctx, canvasRef)
//	pngURL, _ := actions.ToDataURL()  // Default PNG
//	jpegURL, _ := actions.ToDataURL("image/jpeg")
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

// ToDataURLWithQuality returns a data URL with quality setting for lossy formats.
// Quality should be 0.0 to 1.0 (only applies to image/jpeg and image/webp).
//
// Example:
//
//	actions := html.Canvas(ctx, canvasRef)
//	jpegURL, _ := actions.ToDataURLWithQuality("image/jpeg", 0.8)
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

// GetWidth retrieves the width of the canvas in pixels.
func (a *CanvasActions) GetWidth() (int, error) {
	values, err := a.Query("width")
	if err != nil {
		return 0, err
	}
	return toInt(values["width"]), nil
}

// GetHeight retrieves the height of the canvas in pixels.
func (a *CanvasActions) GetHeight() (int, error) {
	values, err := a.Query("height")
	if err != nil {
		return 0, err
	}
	return toInt(values["height"]), nil
}

// SetWidth sets the width of the canvas in pixels.
// Note: This clears the canvas content.
func (a *CanvasActions) SetWidth(width int) {
	a.Set("width", width)
}

// SetHeight sets the height of the canvas in pixels.
// Note: This clears the canvas content.
func (a *CanvasActions) SetHeight(height int) {
	a.Set("height", height)
}
