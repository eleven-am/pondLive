package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// CanvasAPI provides actions and methods for canvas elements.
type CanvasAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewCanvasAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *CanvasAPI[T] {
	return &CanvasAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Async Methods
// ============================================================================

// ToDataURL returns the canvas content as a base64-encoded data URL.
// Format can be "image/png", "image/jpeg", or "image/webp".
// Quality is a number between 0.0 and 1.0 for lossy formats (jpeg, webp).
// This makes a synchronous call to the client and waits for the response.
//
// Example:
//
//	canvasRef := ui.UseElement[*h.CanvasRef](ctx)
//
//	// Export as PNG (lossless)
//	dataURL, err := canvasRef.ToDataURL("image/png", 0)
//
//	// Export as JPEG with 90% quality
//	dataURL, err := canvasRef.ToDataURL("image/jpeg", 0.9)
//
//	return h.Canvas(h.Attach(canvasRef), h.Width("800"), h.Height("600"))
func (a *CanvasAPI[T]) ToDataURL(format string, quality float64) (string, error) {
	if format == "" {
		format = "image/png"
	}
	if quality <= 0 {
		quality = 0.92
	}

	result, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "toDataURL", format, quality)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}

	dataURL, ok := result.(string)
	if !ok {
		return "", nil
	}
	return dataURL, nil
}

// GetContext returns information about the canvas rendering context.
// This makes multiple synchronous calls to the client to gather canvas info.
//
// Example:
//
//	canvasRef := ui.UseElement[*h.CanvasRef](ctx)
//
//	// Get 2D context info (default)
//	info, err := canvasRef.GetContext("2d")
//	if err == nil {
//	    initializeDrawing(info.Width, info.Height)
//	}
//
//	return h.Canvas(h.Attach(canvasRef), h.Width("800"), h.Height("600"))
func (a *CanvasAPI[T]) GetContext(contextType string) (*CanvasContextInfo, error) {
	if contextType == "" {
		contextType = "2d"
	}

	width, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "width")
	if err != nil {
		return nil, err
	}
	height, err := dom.DOMAsyncCall[T](a.ctx, a.ref, "height")
	if err != nil {
		return nil, err
	}

	return &CanvasContextInfo{
		ContextType: contextType,
		Width:       payloadIntDirect(width, 0),
		Height:      payloadIntDirect(height, 0),
	}, nil
}

// payloadIntDirect converts a direct any value to int
func payloadIntDirect(value any, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	if v, ok := value.(float64); ok {
		return int(v)
	}
	return defaultValue
}

// CanvasContextInfo represents information about a canvas rendering context.
type CanvasContextInfo struct {
	ContextType string // "2d", "webgl", "webgl2", "bitmaprenderer"
	Width       int    // Canvas width
	Height      int    // Canvas height
}
