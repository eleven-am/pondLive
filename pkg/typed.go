package pkg

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

type typedEvent[E any] interface {
	*E
	setEvent(work.Event)
}

type shapeField struct {
	index int
	name  string
	opt   bool
	set   func(reflect.Value, any)
}

type eventShape struct {
	fields []shapeField
	lean   []string
	all    []string
}

var shapes sync.Map

func shapeFor[E any]() *eventShape {
	t := reflect.TypeFor[E]()
	if s, ok := shapes.Load(t); ok {
		return s.(*eventShape)
	}
	s := &eventShape{}
	var lean, all []string
	for i := 0; i < t.NumField(); i++ {
		tag, ok := t.Field(i).Tag.Lookup("ev")
		if !ok {
			continue
		}
		name, opt := tag, false
		if n, found := strings.CutSuffix(tag, ",opt"); found {
			name, opt = n, true
		}
		set := setterFor(t.Field(i).Type)
		if set == nil {
			panic(fmt.Sprintf("pkg: event field %s.%s has an unsupported type", t.Name(), t.Field(i).Name))
		}
		s.fields = append(s.fields, shapeField{index: i, name: name, opt: opt, set: set})
		all = append(all, name)
		if !opt {
			lean = append(lean, name)
		}
	}
	if err := protocol.ValidateFields(all...); err != nil {
		panic(err)
	}
	s.lean, s.all = lean, all
	actual, _ := shapes.LoadOrStore(t, s)
	return actual.(*eventShape)
}

func (s *eventShape) decode(dst reflect.Value, payload map[string]any) {
	for _, f := range s.fields {
		if raw, ok := payload[f.name]; ok {
			f.set(dst.Field(f.index), raw)
		}
	}
}

func setterFor(t reflect.Type) func(reflect.Value, any) {
	switch t {
	case reflect.TypeFor[string]():
		return func(v reflect.Value, raw any) {
			if s, ok := raw.(string); ok {
				v.SetString(s)
			}
		}
	case reflect.TypeFor[bool]():
		return func(v reflect.Value, raw any) {
			if b, ok := raw.(bool); ok {
				v.SetBool(b)
			}
		}
	case reflect.TypeFor[int]():
		return func(v reflect.Value, raw any) {
			if n, ok := protocol.Number(raw); ok {
				v.SetInt(int64(n))
			}
		}
	case reflect.TypeFor[float64]():
		return func(v reflect.Value, raw any) {
			if n, ok := protocol.Number(raw); ok {
				v.SetFloat(n)
			}
		}
	case reflect.TypeFor[[]string]():
		return func(v reflect.Value, raw any) { v.Set(reflect.ValueOf(protocol.Strings(raw))) }
	case reflect.TypeFor[[]TouchPoint]():
		return func(v reflect.Value, raw any) {
			list, _ := raw.([]any)
			out := make([]TouchPoint, 0, len(list))
			for _, item := range list {
				if p, ok := item.([]any); ok && len(p) == 3 {
					id, _ := protocol.Number(p[0])
					x, _ := protocol.Number(p[1])
					y, _ := protocol.Number(p[2])
					out = append(out, TouchPoint{ID: int(id), ClientX: x, ClientY: y})
				}
			}
			v.Set(reflect.ValueOf(out))
		}
	case reflect.TypeFor[[]FileMeta]():
		return func(v reflect.Value, raw any) {
			list, _ := raw.([]any)
			out := make([]FileMeta, 0, len(list))
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					name, _ := m[protocol.FileName].(string)
					kind, _ := m[protocol.FileType].(string)
					size, _ := protocol.Number(m[protocol.FileSize])
					mod, _ := protocol.Number(m[protocol.FileModified])
					out = append(out, FileMeta{Name: name, Size: int64(size), Type: kind, LastModified: int64(mod)})
				}
			}
			v.Set(reflect.ValueOf(out))
		}
	}
	return nil
}

func applyOpts(base EventOptions, opts []EventOpt) EventOptions {
	for _, opt := range opts {
		opt.ApplyEvent(&base)
	}
	return base
}

func typedOn[E any, P typedEvent[E]](event string, base EventOptions, h func(E), opts []EventOpt) Item {
	if h == nil {
		return work.Skip()
	}
	shape := shapeFor[E]()
	base.Fields = shape.lean
	o := applyOpts(base, opts)
	if len(o.Fields) == 1 && o.Fields[0] == metadata.FieldsAll {
		o.Fields = shape.all
	}
	if len(o.Keys) > 0 && event != "keydown" && event != "keyup" {
		panic(fmt.Sprintf("pkg: Key applies to keydown and keyup, not %q", event))
	}
	keys := o.Keys
	return OnWith(event, o, func(evt Event) {
		if len(keys) > 0 {
			if k, _ := evt.Payload["key"].(string); !slices.Contains(keys, k) {
				return
			}
		}
		var e E
		P(&e).setEvent(evt)
		shape.decode(reflect.ValueOf(&e).Elem(), evt.Payload)
		h(e)
	})
}
