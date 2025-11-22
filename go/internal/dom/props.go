package dom

import "strings"

// Prop represents an Item that mutates element metadata/attributes
type Prop interface {
	Item
	isProp()
}

// attrProp sets an attribute with a single value (stored as single-element token array)
type attrProp struct{ k, v string }

func (p attrProp) isProp() {}

func (p attrProp) ApplyTo(e *StructuredNode) {
	if e.Attrs == nil {
		e.Attrs = make(map[string][]string)
	}
	e.Attrs[p.k] = []string{p.v}
}

// Attr sets an arbitrary attribute on the element
func Attr(k, v string) Prop { return attrProp{k: k, v: v} }

// Common attribute helpers
func ID(id string) Prop         { return Attr("id", id) }
func Href(url string) Prop      { return Attr("href", url) }
func Src(path string) Prop      { return Attr("src", path) }
func Target(v string) Prop      { return Attr("target", v) }
func Rel(v string) Prop         { return Attr("rel", v) }
func Title(v string) Prop       { return Attr("title", v) }
func Alt(v string) Prop         { return Attr("alt", v) }
func Type(v string) Prop        { return Attr("type", v) }
func Value(v string) Prop       { return Attr("value", v) }
func Name(v string) Prop        { return Attr("name", v) }
func Placeholder(v string) Prop { return Attr("placeholder", v) }
func Data(k, v string) Prop     { return Attr("data-"+k, v) }
func Aria(k, v string) Prop     { return Attr("aria-"+k, v) }

// classProp appends CSS class tokens to the class attribute
type classProp struct{ vals []string }

func (p classProp) isProp() {}

func (p classProp) ApplyTo(e *StructuredNode) {
	if len(p.vals) == 0 {
		return
	}
	if e.Attrs == nil {
		e.Attrs = make(map[string][]string)
	}
	e.Attrs["class"] = append(e.Attrs["class"], p.vals...)
}

// Class appends CSS class tokens to the element
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

// styleProp sets an inline style property
type styleProp struct{ k, v string }

func (p styleProp) isProp() {}

func (p styleProp) ApplyTo(e *StructuredNode) {
	if e.Style == nil {
		e.Style = make(map[string]string)
	}
	e.Style[p.k] = p.v
}

// Style sets an inline CSS property
func Style(k, v string) Prop { return styleProp{k: k, v: v} }

// keyProp sets the key for stable diffing
type keyProp struct{ key string }

func (p keyProp) isProp() {}

func (p keyProp) ApplyTo(e *StructuredNode) {
	e.Key = p.key
}

// GetKey returns the key value, allowing it to be extracted
func (p keyProp) GetKey() string {
	return p.key
}

// Key sets the key for stable diffing in lists
func Key(key string) Prop { return keyProp{key: key} }

// refProp sets the ref ID
type refProp struct{ refID string }

func (p refProp) isProp() {}

func (p refProp) ApplyTo(e *StructuredNode) {
	e.RefID = p.refID
}

// Ref sets the ref ID for this element
func Ref(refID string) Prop { return refProp{refID: refID} }

// handlerProp appends an event handler
type handlerProp struct{ meta HandlerMeta }

func (p handlerProp) isProp() {}

func (p handlerProp) ApplyTo(e *StructuredNode) {
	e.Handlers = append(e.Handlers, p.meta)
}

// Handler adds an event handler to the element
func Handler(event, handlerID string, props ...string) Prop {
	return handlerProp{meta: HandlerMeta{
		Event:   event,
		Handler: handlerID,
		Props:   props,
	}}
}

// routerProp sets router navigation metadata
type routerProp struct{ meta RouterMeta }

func (p routerProp) isProp() {}

func (p routerProp) ApplyTo(e *StructuredNode) {
	e.Router = &p.meta
}

// Router configures router navigation for this element
func Router(path, query, hash, replace string) Prop {
	return routerProp{meta: RouterMeta{
		PathValue: path,
		Query:     query,
		Hash:      hash,
		Replace:   replace,
	}}
}

// Boolean attribute helpers (attributes without values)
type boolAttrProp struct{ k string }

func (p boolAttrProp) isProp() {}

func (p boolAttrProp) ApplyTo(e *StructuredNode) {
	if e.Attrs == nil {
		e.Attrs = make(map[string][]string)
	}

	e.Attrs[p.k] = []string{""}
}

func Disabled() Prop  { return boolAttrProp{k: "disabled"} }
func Checked() Prop   { return boolAttrProp{k: "checked"} }
func Required() Prop  { return boolAttrProp{k: "required"} }
func Readonly() Prop  { return boolAttrProp{k: "readonly"} }
func Autofocus() Prop { return boolAttrProp{k: "autofocus"} }
func Autoplay() Prop  { return boolAttrProp{k: "autoplay"} }
func Controls() Prop  { return boolAttrProp{k: "controls"} }
func Loop() Prop      { return boolAttrProp{k: "loop"} }
func Muted() Prop     { return boolAttrProp{k: "muted"} }
func Selected() Prop  { return boolAttrProp{k: "selected"} }
func Multiple() Prop  { return boolAttrProp{k: "multiple"} }

// attachProp binds an element ref to the element
type attachProp struct {
	target Attachment
}

func (p attachProp) isProp() {}

func (p attachProp) ApplyTo(n *StructuredNode) {
	if p.target != nil {
		p.target.AttachTo(n)
	}
}

// Attach binds an element ref to the element
func Attach(target Attachment) Prop {
	if target == nil {
		return nil
	}
	return attachProp{target: target}
}

type onProp struct {
	event   string
	binding EventBinding
}

func (p onProp) isProp() {}

func (p onProp) ApplyTo(e *StructuredNode) {
	if e.Events == nil {
		e.Events = make(map[string]EventBinding)
	}
	existing, ok := e.Events[p.event]
	if !ok {
		e.Events[p.event] = p.binding
	} else {
		e.Events[p.event] = MergeEventBinding(existing, p.binding)
	}
}

func On(event string, handler EventHandler) Prop {
	binding := EventBinding{Handler: handler}
	binding = binding.WithOptions(DefaultEventOptions(event), event)
	return onProp{event: event, binding: binding}
}

func OnWith(event string, opts EventOptions, handler EventHandler) Prop {
	combined := MergeEventOptions(DefaultEventOptions(event), opts)
	binding := EventBinding{Handler: handler}.WithOptions(combined, event)
	return onProp{event: event, binding: binding}
}

// ExtractKey extracts the first top-level Key from items, returning the key value and remaining items.
// Only scans the top level - nested keys within child elements are not extracted.
func ExtractKey(items []Item) (key string, remaining []Item) {
	for i, item := range items {
		if kp, ok := item.(keyProp); ok {

			remaining = make([]Item, 0, len(items)-1)
			remaining = append(remaining, items[:i]...)
			remaining = append(remaining, items[i+1:]...)
			return kp.GetKey(), remaining
		}
	}

	return "", items
}
