package html2

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type attrItem struct {
	name  string
	value string
}

func (a attrItem) ApplyTo(el *work.Element) {
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs[a.name] = []string{a.value}
}

func Attr(name, value string) Item { return attrItem{name: name, value: value} }

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

func (c classItem) ApplyTo(el *work.Element) {
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

func (s styleItem) ApplyTo(el *work.Element) {
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

func (k keyItem) ApplyTo(el *work.Element) {
	el.Key = k.value
}

func Key(key string) Item {
	return keyItem{value: key}
}

type eventItem struct {
	event   string
	handler work.Handler
}

func (e eventItem) ApplyTo(el *work.Element) {
	if el.Handlers == nil {
		el.Handlers = make(map[string]work.Handler)
	}
	el.Handlers[e.event] = e.handler
}

func On(event string, fn func(work.Event) work.Updates) Item {
	return eventItem{
		event: event,
		handler: work.Handler{
			Fn: fn,
		},
	}
}

func OnWith(event string, options metadata.EventOptions, fn func(work.Event) work.Updates) Item {
	return eventItem{
		event: event,
		handler: work.Handler{
			EventOptions: options,
			Fn:           fn,
		},
	}
}

type attachItem struct {
	ref work.Attachment
}

func (a attachItem) ApplyTo(el *work.Element) {
	el.RefID = a.ref.RefID()
}

func Attach(ref work.Attachment) Item {
	if ref == nil {
		return noopItem{}
	}
	return attachItem{ref: ref}
}

type boolAttrItem struct {
	name string
}

func (b boolAttrItem) ApplyTo(el *work.Element) {
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

func (u unsafeHTMLItem) ApplyTo(el *work.Element) {
	el.UnsafeHTML = u.html
}

func UnsafeHTML(html string) Item {
	return unsafeHTMLItem{html: html}
}
