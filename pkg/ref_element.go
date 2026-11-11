package pkg

import (
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
)

type DOMRect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

type ScrollMetrics struct {
	ScrollTop    float64
	ScrollLeft   float64
	ScrollHeight float64
	ScrollWidth  float64
	ClientHeight float64
	ClientWidth  float64
}

type ScrollOptions struct {
	Behavior string
	Block    string
	Inline   string
}

func decodeRect(v any) (DOMRect, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return DOMRect{}, false
	}
	f := protocol.Floats(m, protocol.RectKeys)
	return DOMRect{X: f[0], Y: f[1], Width: f[2], Height: f[3], Top: f[4], Right: f[5], Bottom: f[6], Left: f[7]}, true
}

func decodeScroll(v any) (ScrollMetrics, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return ScrollMetrics{}, false
	}
	f := protocol.Floats(m, protocol.ScrollKeys)
	return ScrollMetrics{ScrollTop: f[0], ScrollLeft: f[1], ScrollHeight: f[2], ScrollWidth: f[3], ClientHeight: f[4], ClientWidth: f[5]}, true
}

func (r *ElementRef) Focus() { r.ref.Call("focus") }

func (r *ElementRef) Blur() { r.ref.Call("blur") }

func (r *ElementRef) Click() { r.ref.Call("click") }

func (r *ElementRef) ScrollIntoView(opts *ScrollOptions) {
	if opts == nil {
		r.ref.Call("scrollIntoView")
		return
	}
	r.ref.Call("scrollIntoView", map[string]any{"behavior": opts.Behavior, "block": opts.Block, "inline": opts.Inline})
}

func (r *ElementRef) SetScrollTop(value float64) { r.ref.Set("scrollTop", value) }

func (r *ElementRef) SetScrollLeft(value float64) { r.ref.Set("scrollLeft", value) }

func (r *ElementRef) Focused(opts ...ObserveOpt) ReadSignal[bool] {
	return live(r, boolSpec("focused", false, hasAttr("autofocus")), opts)
}

func (r *ElementRef) Rect(opts ...ObserveOpt) ReadSignal[DOMRect] {
	return live(r, runtime.RefSpec[DOMRect]{Prop: "rect", Decode: decodeRect}, opts)
}

func (r *ElementRef) Scroll(opts ...ObserveOpt) ReadSignal[ScrollMetrics] {
	return live(r, runtime.RefSpec[ScrollMetrics]{Prop: "scroll", Decode: decodeScroll}, opts)
}

func (r *ElementRef) ComputedStyle(fn func(map[string]string, error), properties ...string) {
	result(r, "getComputedStyle", func(v any) map[string]string {
		styles, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		out := make(map[string]string, len(styles))
		for key, val := range styles {
			if s, ok := val.(string); ok {
				out[key] = s
			}
		}
		return out
	}, fn, properties)
}

func (r *ElementRef) CheckVisibility(fn func(bool, error)) {
	result(r, "checkVisibility", asBool, fn)
}

func (r *ElementRef) Matches(selector string, fn func(bool, error)) {
	result(r, "matches", asBool, fn, selector)
}
