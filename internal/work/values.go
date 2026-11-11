package work

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/reactive"
)

type fmtSource[T any] struct {
	r reactive.Readable[T]
}

func (f fmtSource[T]) Get() string  { return fmt.Sprint(f.r.Get()) }
func (f fmtSource[T]) Peek() string { return fmt.Sprint(f.r.Peek()) }

type intSource struct {
	r reactive.Readable[int]
}

func (f intSource) Get() string  { return strconv.Itoa(f.r.Get()) }
func (f intSource) Peek() string { return strconv.Itoa(f.r.Peek()) }

type reflectSource struct {
	get, peek reflect.Value
}

func (f reflectSource) Get() string  { return fmt.Sprint(f.get.Call(nil)[0].Interface()) }
func (f reflectSource) Peek() string { return fmt.Sprint(f.peek.Call(nil)[0].Interface()) }

type UnsupportedValueError struct {
	Type string
}

func (e *UnsupportedValueError) Error() string {
	return "work: unsupported value type " + e.Type
}

func resolveString(v any) (static string, src reactive.Readable[string]) {
	switch x := v.(type) {
	case string:
		return x, nil
	case reactive.Readable[string]:
		return "", x
	case reactive.Readable[int]:
		return "", intSource{x}
	case reactive.Readable[bool]:
		return "", fmtSource[bool]{x}
	case reactive.Readable[int64]:
		return "", fmtSource[int64]{x}
	case reactive.Readable[float64]:
		return "", fmtSource[float64]{x}
	case int:
		return strconv.Itoa(x), nil
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, bool:
		return fmt.Sprint(x), nil
	}
	if v != nil {
		rv := reflect.ValueOf(v)
		get, peek := rv.MethodByName("Get"), rv.MethodByName("Peek")
		if get.IsValid() && peek.IsValid() && get.Type().NumIn() == 0 && get.Type().NumOut() == 1 &&
			peek.Type().NumIn() == 0 && peek.Type().NumOut() == 1 {
			return "", reflectSource{get: get, peek: peek}
		}
	}
	panic(&UnsupportedValueError{Type: fmt.Sprintf("%T", v)})
}

func TextOf[T any](v T) Node {
	static, src := resolveString(any(v))
	if src != nil {
		return &BoundText{Source: src}
	}
	return &Text{Value: static}
}

type attrItem struct{ attr Attr }

func (a attrItem) applyTo(el *Element) {
	for i := range el.Attrs {
		if el.Attrs[i].Name == a.attr.Name {
			el.Attrs[i] = a.attr
			return
		}
	}
	el.Attrs = append(el.Attrs, a.attr)
}

type AttrOpt func(*attrOpts)

type attrOpts struct{ omitEmpty bool }

func OmitEmpty() AttrOpt { return func(o *attrOpts) { o.omitEmpty = true } }

func Attribute[T any](name string, v T, opts ...AttrOpt) Item {
	var o attrOpts
	for _, opt := range opts {
		opt(&o)
	}
	if o.omitEmpty {
		return omitEmptyAttribute(name, v)
	}
	switch x := any(v).(type) {
	case bool:
		if x {
			return attrItem{Attr{Name: name, Kind: AttrPresent}}
		}
		return attrItem{Attr{Name: name, Kind: AttrAbsent}}
	case reactive.Readable[bool]:
		return attrItem{Attr{Name: name, Kind: AttrDynamicPresence, Presence: x}}
	}
	static, src := resolveString(any(v))
	if src != nil {
		return attrItem{Attr{Name: name, Kind: AttrDynamic, Source: src}}
	}
	return attrItem{Attr{Name: name, Kind: AttrStatic, Value: static}}
}

func omitEmptyAttribute[T any](name string, v T) Item {
	static, src := resolveString(any(v))
	switch {
	case src != nil:
		return attrItem{Attr{Name: name, Kind: AttrDynamicOptional, Source: src}}
	case static == "":
		return attrItem{Attr{Name: name, Kind: AttrAbsent}}
	}
	return attrItem{Attr{Name: name, Kind: AttrStatic, Value: static}}
}

type classItem struct{ parts []ClassPart }

func (c classItem) applyTo(el *Element) { el.Classes = append(el.Classes, c.parts...) }

func Class(parts ...any) Item {
	out := make([]ClassPart, 0, len(parts))
	for _, p := range parts {
		switch x := p.(type) {
		case nil:
		case ClassPart:
			out = append(out, x)
		case reactive.Readable[string]:
			out = append(out, ClassPart{Source: x})
		case string:
			if !strings.ContainsAny(x, " \t\n\r") {
				if x != "" {
					out = append(out, ClassPart{Static: x})
				}
				continue
			}
			for _, token := range strings.Fields(x) {
				out = append(out, ClassPart{Static: token})
			}
		default:
			panic(&UnsupportedValueError{Type: fmt.Sprintf("%T", p)})
		}
	}
	return classItem{parts: out}
}

type styleItem struct{ decl StyleDecl }

func (s styleItem) applyTo(el *Element) {
	for i := range el.Styles {
		if el.Styles[i].Prop == s.decl.Prop {
			el.Styles[i] = s.decl
			return
		}
	}
	el.Styles = append(el.Styles, s.decl)
}

func Style[T any](prop string, v T) Item {
	static, src := resolveString(any(v))
	return styleItem{StyleDecl{Prop: prop, Value: static, Source: src}}
}

type stylesItem []Item

func (s stylesItem) applyTo(el *Element) {
	for _, it := range s {
		it.applyTo(el)
	}
}

func Styles[V any](styles map[string]V) Item {
	props := make([]string, 0, len(styles))
	for p := range styles {
		props = append(props, p)
	}
	slices.Sort(props)
	out := make(stylesItem, 0, len(props))
	for _, p := range props {
		out = append(out, Style(p, styles[p]))
	}
	return out
}

type rawItem struct {
	raw string
	src reactive.Readable[string]
}

func (r rawItem) applyTo(el *Element) {
	if r.src != nil {
		el.Raw = &Raw{Source: r.src}
		return
	}
	el.Raw = &Raw{Static: r.raw}
}

func UnsafeHTML[T any](v T) Item {
	static, src := resolveString(any(v))
	return rawItem{raw: static, src: src}
}

type eventItem struct {
	event   string
	options metadata.EventOptions
	fn      func(Event)
}

func (e eventItem) applyTo(el *Element) {
	for i := range el.Events {
		b := &el.Events[i]
		if b.Event == e.event {
			b.Options = metadata.MergeHandlers(b.Options, e.options)
			b.Fns = append(b.Fns, e.fn)
			return
		}
	}
	el.Events = append(el.Events, EventBinding{Event: e.event, Options: e.options, Fns: []func(Event){e.fn}})
}

func On(event string, fn func(Event)) Item {
	return OnWith(event, metadata.EventOptions{}, fn)
}

func OnWith(event string, options metadata.EventOptions, fn func(Event)) Item {
	return eventItem{event: event, options: options, fn: fn}
}
