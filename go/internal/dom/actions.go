package dom

import "strings"

const (
	domEffectType   = "dom"
	domActionCall   = "dom.call"
	domActionSet    = "dom.set"
	domActionToggle = "dom.toggle"
	domActionClass  = "dom.class"
	domActionScroll = "dom.scroll"
)

// DOMCall instructs the client to invoke a method on the referenced element.
func DOMCall[T ElementDescriptor](ctx Dispatcher, ref *ElementRef[T], method string, args ...any) {
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
func DOMSet[T ElementDescriptor](ctx Dispatcher, ref *ElementRef[T], prop string, value any) {
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
func DOMToggle[T ElementDescriptor](ctx Dispatcher, ref *ElementRef[T], prop string, on bool) {
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
func DOMToggleClass[T ElementDescriptor](ctx Dispatcher, ref *ElementRef[T], class string, on bool) {
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
func DOMScrollIntoView[T ElementDescriptor](ctx Dispatcher, ref *ElementRef[T], opts ScrollOptions) {
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

func enqueueDOMAction[T ElementDescriptor](ctx Dispatcher, ref *ElementRef[T], build func(*DOMActionEffect) bool) {
	if ctx == nil {
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
	ctx.EnqueueDOMAction(effect)
}

func boolRef(v bool) *bool {
	b := v
	return &b
}
