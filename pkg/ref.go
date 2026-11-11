package pkg

import (
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
)

type ObserveOpt = runtime.ObserveOpt

type ElementRef struct{ ref *runtime.Ref }

func (r *ElementRef) Attached() ReadSignal[bool] { return r.ref.AttachedSignal() }

type InputRef struct{ *ElementRef }

type FormRef struct{ *ElementRef }

type MediaRef struct{ *ElementRef }

type DialogRef struct{ *ElementRef }

type CanvasRef struct{ *ElementRef }

type DetailsRef struct{ *ElementRef }

type RefType interface {
	ElementRef | InputRef | FormRef | MediaRef | DialogRef | CanvasRef | DetailsRef
}

func UseRef[T RefType](ctx *Ctx) *T {
	return runtime.UseCell(ctx, func() *T {
		t := new(T)
		switch p := any(t).(type) {
		case *ElementRef:
			p.ref = runtime.NewRef(ctx, "ElementRef", nil)
		case *InputRef:
			p.ElementRef = newElementRef(ctx, "InputRef", "input", "textarea", "select", "button", "fieldset", "meter", "progress", "option", "optgroup")
		case *FormRef:
			p.ElementRef = newElementRef(ctx, "FormRef", "form")
		case *MediaRef:
			p.ElementRef = newElementRef(ctx, "MediaRef", "audio", "video")
		case *DialogRef:
			p.ElementRef = newElementRef(ctx, "DialogRef", "dialog")
		case *CanvasRef:
			p.ElementRef = newElementRef(ctx, "CanvasRef", "canvas")
		case *DetailsRef:
			p.ElementRef = newElementRef(ctx, "DetailsRef", "details")
		}
		return t
	})
}

func newElementRef(ctx *Ctx, name string, tags ...string) *ElementRef {
	allowed := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		allowed[tag] = struct{}{}
	}
	return &ElementRef{ref: runtime.NewRef(ctx, name, func(tag string) bool {
		_, ok := allowed[tag]
		return ok
	})}
}

func live[T any](r *ElementRef, spec runtime.RefSpec[T], opts []ObserveOpt) ReadSignal[T] {
	return runtime.RefSignal(r.ref, spec, opts...)
}

func property[T any](r *ElementRef, prop string, conv func(any) T, fn func(T, error)) {
	if fn == nil {
		return
	}
	r.ref.Query(func(values map[string]any, err error) {
		if err != nil {
			var zero T
			fn(zero, err)
			return
		}
		fn(conv(values[prop]), nil)
	}, prop)
}

func result[T any](r *ElementRef, method string, conv func(any) T, fn func(T, error), args ...any) {
	if fn == nil {
		return
	}
	r.ref.CallThen(func(res any, err error) {
		if err != nil {
			var zero T
			fn(zero, err)
			return
		}
		fn(conv(res), nil)
	}, method, args...)
}

func decodeOf[T any](conv func(any) T, ok func(any) bool) func(any) (T, bool) {
	return func(v any) (T, bool) {
		if !ok(v) {
			var zero T
			return zero, false
		}
		return conv(v), true
	}
}

func isBool(v any) bool   { _, ok := v.(bool); return ok }
func isString(v any) bool { _, ok := v.(string); return ok }
func isNumber(v any) bool {
	_, ok := protocol.Number(v)
	return v == nil || ok
}
func isMap(v any) bool { _, ok := v.(map[string]any); return ok }

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func toFloat64(v any) float64 {
	n, _ := protocol.Number(v)
	return n
}

func toInt(v any) int { return int(toFloat64(v)) }

func hasAttr(name string) func(map[string]string) (bool, bool) {
	return func(attrs map[string]string) (bool, bool) {
		_, present := attrs[name]
		return present, attrs != nil
	}
}

func attrValue(name string) func(map[string]string) (string, bool) {
	return func(attrs map[string]string) (string, bool) {
		v, ok := attrs[name]
		return v, ok
	}
}

var (
	decodeBool   = decodeOf(asBool, isBool)
	decodeString = decodeOf(asString, isString)
	decodeFloat  = decodeOf(toFloat64, isNumber)
	decodeInt    = decodeOf(toInt, isNumber)
)

func boolSpec(prop string, zero bool, seed func(map[string]string) (bool, bool)) runtime.RefSpec[bool] {
	return runtime.RefSpec[bool]{Prop: prop, Zero: zero, Decode: decodeBool, Seed: seed}
}

func floatSpec(prop string, zero float64) runtime.RefSpec[float64] {
	return runtime.RefSpec[float64]{Prop: prop, Zero: zero, Decode: decodeFloat}
}
