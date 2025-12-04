package work

import (
	"strings"

	"github.com/eleven-am/pondlive/internal/metadata"
)

type attrItem struct {
	name  string
	value string
}

func (a attrItem) ApplyTo(el *Element) {
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs[a.name] = []string{a.value}
}

func Attr(name, value string) Item {
	return attrItem{name: name, value: value}
}

func ID(id string) Item         { return Attr("id", id) }
func Href(url string) Item      { return Attr("href", url) }
func Src(path string) Item      { return Attr("src", path) }
func Target(v string) Item      { return Attr("target", v) }
func Rel(v string) Item         { return Attr("rel", v) }
func Title(v string) Item       { return Attr("title", v) }
func Alt(v string) Item         { return Attr("alt", v) }
func Type(v string) Item        { return Attr("type", v) }
func Value(v string) Item       { return Attr("value", v) }
func Name(v string) Item        { return Attr("name", v) }
func Placeholder(v string) Item { return Attr("placeholder", v) }
func Data(k, v string) Item     { return Attr("data-"+k, v) }
func Aria(k, v string) Item     { return Attr("aria-"+k, v) }

type classItem struct {
	vals []string
}

func (c classItem) ApplyTo(el *Element) {
	if len(c.vals) == 0 {
		return
	}
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs["class"] = append(el.Attrs["class"], c.vals...)
}

func Class(classes ...string) Item {
	filtered := make([]string, 0, len(classes))
	for _, c := range classes {
		token := strings.TrimSpace(c)
		if token == "" {
			continue
		}
		filtered = append(filtered, token)
	}
	return classItem{vals: filtered}
}

type styleItem struct {
	property string
	value    string
}

func (s styleItem) ApplyTo(el *Element) {
	if el.Style == nil {
		el.Style = make(map[string]string)
	}
	el.Style[s.property] = s.value
}

func Style(property, value string) Item {
	return styleItem{property: property, value: value}
}

type keyItem struct {
	value string
}

func (k keyItem) ApplyTo(el *Element) {
	el.Key = k.value
}

func Key(key string) Item {
	return keyItem{value: key}
}

type eventItem struct {
	event   string
	handler Handler
}

func (e eventItem) ApplyTo(el *Element) {
	if el.Handlers == nil {
		el.Handlers = make(map[string]Handler)
	}

	existing, exists := el.Handlers[e.event]
	if !exists {
		el.Handlers[e.event] = e.handler
		return
	}

	oldFn := existing.Fn
	newFn := e.handler.Fn

	el.Handlers[e.event] = Handler{
		EventOptions: MergeEventOptions(existing.EventOptions, e.handler.EventOptions),
		Fn: func(evt Event) Updates {
			var oldResult, newResult Updates
			if oldFn != nil {
				oldResult = oldFn(evt)
			}
			if newFn != nil {
				newResult = newFn(evt)
			}
			if newResult != nil {
				return newResult
			}
			return oldResult
		},
	}
}

func On(event string, fn func(Event) Updates) Item {
	return OnWith(event, metadata.EventOptions{}, fn)
}

func OnWith(event string, options metadata.EventOptions, fn func(Event) Updates) Item {
	return eventItem{
		event: event,
		handler: Handler{
			EventOptions: options,
			Fn:           fn,
		},
	}
}

type attachItem struct {
	ref Attachment
}

func (a attachItem) ApplyTo(el *Element) {
	a.ref.AttachTo(el)
}

func Attach(ref Attachment) Item {
	if ref == nil {
		return noopItem{}
	}
	return attachItem{ref: ref}
}

type boolAttrItem struct {
	name string
}

func (b boolAttrItem) ApplyTo(el *Element) {
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs[b.name] = []string{""}
}

func Disabled() Item  { return boolAttrItem{name: "disabled"} }
func Checked() Item   { return boolAttrItem{name: "checked"} }
func Required() Item  { return boolAttrItem{name: "required"} }
func Readonly() Item  { return boolAttrItem{name: "readonly"} }
func Autofocus() Item { return boolAttrItem{name: "autofocus"} }
func Autoplay() Item  { return boolAttrItem{name: "autoplay"} }
func Controls() Item  { return boolAttrItem{name: "controls"} }
func Loop() Item      { return boolAttrItem{name: "loop"} }
func Muted() Item     { return boolAttrItem{name: "muted"} }
func Selected() Item  { return boolAttrItem{name: "selected"} }
func Multiple() Item  { return boolAttrItem{name: "multiple"} }

type unsafeHTMLItem struct {
	html string
}

func (u unsafeHTMLItem) ApplyTo(el *Element) {
	el.UnsafeHTML = u.html
}

func UnsafeHTML(html string) Item {
	return unsafeHTMLItem{html: html}
}

func MergeEventOptions(a, b metadata.EventOptions) metadata.EventOptions {
	merged := metadata.EventOptions{
		Prevent: a.Prevent || b.Prevent,
		Stop:    a.Stop || b.Stop,
		Passive: a.Passive || b.Passive,
		Once:    a.Once || b.Once,
		Capture: a.Capture || b.Capture,
	}

	if a.Debounce > 0 && b.Debounce > 0 {
		if a.Debounce < b.Debounce {
			merged.Debounce = a.Debounce
		} else {
			merged.Debounce = b.Debounce
		}
	} else if a.Debounce > 0 {
		merged.Debounce = a.Debounce
	} else {
		merged.Debounce = b.Debounce
	}

	if a.Throttle > 0 && b.Throttle > 0 {
		if a.Throttle < b.Throttle {
			merged.Throttle = a.Throttle
		} else {
			merged.Throttle = b.Throttle
		}
	} else if a.Throttle > 0 {
		merged.Throttle = a.Throttle
	} else {
		merged.Throttle = b.Throttle
	}

	merged.Props = deduplicateStrings(append(a.Props, b.Props...))
	merged.Listen = deduplicateStrings(append(a.Listen, b.Listen...))

	return merged
}

func mergeUpdates(a, b Updates) Updates {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	aSlice, aIsSlice := a.([]Updates)
	bSlice, bIsSlice := b.([]Updates)
	switch {
	case aIsSlice && bIsSlice:
		return append(aSlice, bSlice...)
	case aIsSlice:
		return append(aSlice, b)
	case bIsSlice:
		return append([]Updates{a}, bSlice...)
	default:
		return []Updates{a, b}
	}
}

func deduplicateStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, s := range input {
		if s == "" {
			continue
		}
		if _, exists := seen[s]; exists {
			continue
		}
		seen[s] = struct{}{}
		result = append(result, s)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
