package html

import "strings"

type attrProp struct{ k, v string }

func (p attrProp) isProp() {}

func (p attrProp) ApplyTo(e *Element) {
	if e.Attrs == nil {
		e.Attrs = map[string]string{}
	}
	e.Attrs[p.k] = p.v
}

type mutableAttrProp struct{ attrProp }

func (p mutableAttrProp) ApplyTo(e *Element) {
	p.attrProp.ApplyTo(e)
	if e.MutableAttrs == nil {
		e.MutableAttrs = map[string]bool{}
	}
	e.MutableAttrs[p.k] = true
}

// Attr sets an arbitrary attribute on the element.
func Attr(k, v string) Prop { return attrProp{k: k, v: v} }

// MutableAttr sets an attribute and marks it as dynamic so it renders via a dynamic slot immediately.
func MutableAttr(k, v string) Prop { return mutableAttrProp{attrProp{k: k, v: v}} }

// ID sets the id attribute.
func ID(id string) Prop { return Attr("id", id) }

// Href sets the href attribute.
func Href(url string) Prop { return Attr("href", url) }

// Src sets the src attribute.
func Src(path string) Prop { return Attr("src", path) }

// Target sets the target attribute.
func Target(v string) Prop { return Attr("target", v) }

// Rel sets the rel attribute.
func Rel(v string) Prop { return Attr("rel", v) }

// Title sets the title attribute.
func Title(v string) Prop { return Attr("title", v) }

// Alt sets the alt attribute.
func Alt(v string) Prop { return Attr("alt", v) }

// Type sets the type attribute.
func Type(v string) Prop { return Attr("type", v) }

// Value sets the value attribute.
func Value(v string) Prop { return Attr("value", v) }

// Name sets the name attribute.
func Name(v string) Prop { return Attr("name", v) }

// Data sets a data-* attribute.
func Data(k, v string) Prop { return Attr("data-"+k, v) }

// Aria sets an aria-* attribute.
func Aria(k, v string) Prop { return Attr("aria-"+k, v) }

type classProp struct{ vals []string }

func (p classProp) isProp() {}

func (p classProp) ApplyTo(e *Element) {
	if len(p.vals) == 0 {
		return
	}
	e.Class = append(e.Class, p.vals...)
}

// Class appends CSS class tokens to the element.
func Class(classes ...string) Prop {
	filtered := make([]string, 0, len(classes))
	for _, c := range classes {
		token := strings.TrimSpace(c)
		if token == "" {
			continue
		}
		filtered = append(filtered, token)
	}
	return classProp{vals: filtered}
}

type styleProp struct{ k, v string }

func (p styleProp) isProp() {}

func (p styleProp) ApplyTo(e *Element) {
	if e.Style == nil {
		e.Style = map[string]string{}
	}
	e.Style[p.k] = p.v
}

// Style sets an inline style declaration.
func Style(k, v string) Prop { return styleProp{k: k, v: v} }

type keyProp struct{ key string }

func (p keyProp) isProp() {}

func (p keyProp) ApplyTo(e *Element) { e.Key = p.key }

// Key assigns a stable identity for keyed lists.
func Key(k string) Prop { return keyProp{key: k} }

type rawHTMLProp struct{ html string }

func (p rawHTMLProp) isProp() {}

func (p rawHTMLProp) ApplyTo(e *Element) { e.Unsafe = &p.html }

// UnsafeHTML sets pre-escaped inner HTML for the element.
func UnsafeHTML(html string) Prop { return rawHTMLProp{html: html} }

type onProp struct {
	event   string
	binding EventBinding
}

func (p onProp) isProp() {}

func (p onProp) ApplyTo(e *Element) {
	if e.Events == nil {
		e.Events = map[string]EventBinding{}
	}
	if existing, ok := e.Events[p.event]; ok {
		e.Events[p.event] = mergeEventBinding(existing, p.binding)
		return
	}
	e.Events[p.event] = p.binding
}

// On attaches an event handler for the named DOM event.
//
// Handlers registered this way are deduplicated by their function identity so
// that reusing the same function value across renders keeps a stable handler
// ID. When capturing values in closures, keep in mind that Go may reuse the
// same function pointer for identical closure bodies; if you need per-instance
// isolation, you can either attach a ref and register listeners through it, or
// use OnWith with a custom Key to override deduplication.
func On(event string, handler EventHandler) Prop {
	binding := EventBinding{Handler: handler}
	binding = binding.WithOptions(defaultEventOptions(event), event)
	return onProp{event: event, binding: binding}
}

// OnWith attaches an event handler together with additional options that
// describe which DOM events should be listened to and which properties should
// be captured when the event fires. Handler deduplication obeys the same
// function-identity semantics described in On. To override pointer-based
// deduplication for closures with identical code but different captured values,
// provide a custom Key in the EventOptions.
func OnWith(event string, opts EventOptions, handler EventHandler) Prop {
	combined := mergeEventOptions(defaultEventOptions(event), opts)
	binding := EventBinding{Handler: handler}.WithOptions(combined, event)
	return onProp{event: event, binding: binding}
}
