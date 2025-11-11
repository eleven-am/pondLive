package runtime

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	html "github.com/eleven-am/pondlive/go/pkg/live/html"
)

const (
	domEffectType   = "dom"
	domActionCall   = "dom.call"
	domActionSet    = "dom.set"
	domActionToggle = "dom.toggle"
	domActionClass  = "dom.class"
	domActionScroll = "dom.scroll"
)

// ScrollOptions configure how the browser should scroll the referenced element
// into view.
type ScrollOptions struct {
	Behavior string
	Block    string
	Inline   string
}

// DOMCall instructs the client to invoke a method on the referenced element.
func DOMCall[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], method string, args ...any) {
	method = strings.TrimSpace(method)
	if method == "" {
		return
	}
	enqueueDOMAction(ctx, ref, func(effect *DOMActionEffect) bool {
		effect.Kind = domActionCall
		effect.Method = method
		if len(args) > 0 {
			effect.Args = append([]any(nil), args...)
		}
		return true
	})
}

// DOMSet assigns the provided value to the named property on the referenced element.
func DOMSet[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], prop string, value any) {
	prop = strings.TrimSpace(prop)
	if prop == "" {
		return
	}
	enqueueDOMAction(ctx, ref, func(effect *DOMActionEffect) bool {
		effect.Kind = domActionSet
		effect.Prop = prop
		effect.Value = value
		return true
	})
}

// DOMToggle updates a boolean property on the referenced element.
func DOMToggle[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], prop string, on bool) {
	prop = strings.TrimSpace(prop)
	if prop == "" {
		return
	}
	enqueueDOMAction(ctx, ref, func(effect *DOMActionEffect) bool {
		effect.Kind = domActionToggle
		effect.Prop = prop
		effect.Value = on
		return true
	})
}

// DOMToggleClass toggles the provided class on the referenced element.
func DOMToggleClass[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], class string, on bool) {
	class = strings.TrimSpace(class)
	if class == "" {
		return
	}
	enqueueDOMAction(ctx, ref, func(effect *DOMActionEffect) bool {
		effect.Kind = domActionClass
		effect.Class = class
		effect.On = boolRef(on)
		return true
	})
}

// DOMScrollIntoView scrolls the referenced element into view using the provided options.
func DOMScrollIntoView[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], opts ScrollOptions) {
	enqueueDOMAction(ctx, ref, func(effect *DOMActionEffect) bool {
		effect.Kind = domActionScroll
		if trimmed := strings.TrimSpace(opts.Behavior); trimmed != "" {
			effect.Behavior = trimmed
		}
		if trimmed := strings.TrimSpace(opts.Block); trimmed != "" {
			effect.Block = trimmed
		}
		if trimmed := strings.TrimSpace(opts.Inline); trimmed != "" {
			effect.Inline = trimmed
		}
		return true
	})
}

func enqueueDOMAction[T dom.ElementDescriptor](ctx Ctx, ref *html.ElementRef[T], build func(*DOMActionEffect) bool) {
	if ctx.sess == nil || ctx.sess.owner == nil {
		return
	}
	if ref == nil {
		return
	}
	refID := strings.TrimSpace(ref.ID())
	if refID == "" {
		return
	}
	effect := DOMActionEffect{
		Type: domEffectType,
		Ref:  refID,
	}
	if !build(&effect) {
		return
	}
	ctx.sess.owner.enqueueFrameEffect(effect)
}

func boolRef(v bool) *bool {
	b := v
	return &b
}
