package runtime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

const (
	WarnBindReadOnly WarningCode = "BIND_READ_ONLY_SIGNAL"
	WarnBindMismatch WarningCode = "BIND_TYPE_MISMATCH"

	selectionAttr = protocol.AttrSelection
)

type bindTarget uint8

const (
	targetAttr bindTarget = iota
	targetValue
	targetChecked
	targetSelection
	targetOpen
)

type BindMismatchError struct {
	Tag, Type string
	Got       work.BindKind
	Want      work.BindKind
}

func (e *BindMismatchError) Error() string {
	return fmt.Sprintf("runtime: Bind on <%s%s> needs a %s, got a %s", e.Tag, e.Type, e.Want, e.Got)
}

type BindOwnedError struct{ Tag, Name string }

func (e *BindOwnedError) Error() string {
	return fmt.Sprintf("runtime: <%s %s> is state the user can change; set it with Bind, not Attr", e.Tag, e.Name)
}

func bindOwned(el *work.Element, name string) bool {
	switch strings.ToLower(name) {
	case "value":
		t, _ := classifyBind(el)
		return t == targetValue
	case "checked":
		t, _ := classifyBind(el)
		return t == targetChecked
	case "selected":
		return el.Tag == "option"
	case "open":
		return el.Tag == "details" || el.Tag == "dialog"
	}
	return false
}

func (e *emitter) checkBindOwned(sc *scope, el *work.Element) {
	for _, a := range el.Attrs {
		if !bindOwned(el, a.Name) {
			continue
		}
		err := &BindOwnedError{Tag: el.Tag, Name: a.Name}
		if e.s.devMode {
			panic(err)
		}
		e.s.Report(sc.o, Report{Severity: SeverityDiagnostic, Phase: "attribute:bind-owned", Message: err.Error(), Meta: map[string]any{"tag": el.Tag, "attribute": a.Name}})
	}
}

type echoKey struct {
	id   uint32
	name string
}

type selectState struct{ values map[string]struct{} }

type jsonList struct{ src reactive.Readable[[]string] }

func (j jsonList) Get() string  { return encodeList(j.src.Get()) }
func (j jsonList) Peek() string { return encodeList(j.src.Peek()) }

func encodeList(v []string) string {
	if v == nil {
		v = []string{}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func attrState(el *work.Element, name string) (value string, present bool) {
	for _, a := range el.Attrs {
		if a.Name != name {
			continue
		}
		switch a.Kind {
		case work.AttrStatic:
			return a.Value, true
		case work.AttrPresent:
			return "", true
		case work.AttrDynamic, work.AttrDynamicOptional:
			v := peekString(a.Source)
			return v, v != ""
		case work.AttrDynamicPresence:
			if peekBool(a.Presence) {
				return "", true
			}
		}
	}
	return "", false
}

func classifyBind(el *work.Element) (bindTarget, string) {
	switch el.Tag {
	case "input":
		typ, _ := attrState(el, "type")
		typ = strings.ToLower(typ)
		switch typ {
		case "checkbox", "radio":
			return targetChecked, typ
		case "hidden", "button", "submit", "reset", "image", "file":
			return targetAttr, typ
		}
		return targetValue, typ
	case "textarea":
		return targetValue, ""
	case "details", "dialog":
		return targetOpen, ""
	case "select":
		if _, multiple := attrState(el, "multiple"); multiple {
			return targetSelection, ""
		}
		return targetValue, ""
	}
	return targetAttr, ""
}

func (t bindTarget) wants() work.BindKind {
	switch t {
	case targetChecked, targetOpen:
		return work.BindBool
	case targetSelection:
		return work.BindStrings
	}
	return work.BindString
}

func (e *emitter) applyBind(sc *scope, el *work.Element, id uint32) (body *string, restore func()) {
	spec := el.Bind
	el.Bind = nil
	target, typ := classifyBind(el)
	if spec.Kind != target.wants() {
		suffix := ""
		if typ != "" {
			suffix = " type=" + typ
		}
		err := &BindMismatchError{Tag: el.Tag, Type: suffix, Got: spec.Kind, Want: target.wants()}
		e.s.attachProblem(sc.o, WarnBindMismatch, err)
		return nil, nil
	}
	_, readonly := attrState(el, "readonly")
	_, disabled := attrState(el, "disabled")
	editable := target != targetAttr && !readonly && !disabled

	switch target {
	case targetChecked, targetOpen:
		name := "checked"
		if target == targetOpen {
			name = "open"
		}
		switch {
		case spec.ReadBool != nil:
			el.Attrs = append(el.Attrs, work.Attr{Name: name, Kind: work.AttrDynamicPresence, Presence: spec.ReadBool})
		case spec.ConstBool:
			el.Attrs = append(el.Attrs, work.Attr{Name: name, Kind: work.AttrPresent})
		}
	case targetSelection:
		var src reactive.Readable[string]
		var initial []string
		if spec.ReadStrings != nil {
			src, initial = jsonList{spec.ReadStrings}, peekStrings(spec.ReadStrings)
		} else {
			initial = spec.ConstStrings
			el.Attrs = append(el.Attrs, work.Attr{Name: selectionAttr, Kind: work.AttrStatic, Value: encodeList(initial)})
		}
		if src != nil {
			el.Attrs = append(el.Attrs, work.Attr{Name: selectionAttr, Kind: work.AttrDynamic, Source: src})
		}
		restore = e.enterSelect(initial)
	default:
		var current string
		if spec.ReadString != nil {
			current = peekString(spec.ReadString)
			el.Attrs = append(el.Attrs, work.Attr{Name: "value", Kind: work.AttrDynamic, Source: spec.ReadString})
		} else {
			current = spec.ConstString
			el.Attrs = append(el.Attrs, work.Attr{Name: "value", Kind: work.AttrStatic, Value: spec.ConstString})
		}
		if el.Tag == "textarea" {
			body = &current
		}
		if el.Tag == "select" {
			restore = e.enterSelect([]string{current})
		}
	}

	if !editable {
		return body, restore
	}
	if spec.Writable() {
		el.Events = append(e.s.bindEvents(el, spec, target, id), el.Events...)
	} else if !spec.Constant() {
		e.s.Report(sc.o, Report{
			Severity: SeverityWarning,
			Code:     string(WarnBindReadOnly),
			Message:  "typing here won't update this value; pass a writable Signal to Bind for two-way",
			Meta:     map[string]any{"tag": el.Tag},
		})
	}
	return body, restore
}

func (e *emitter) enterSelect(vals []string) func() {
	prev := e.sel
	set := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		set[v] = struct{}{}
	}
	e.sel = &selectState{values: set}
	return func() { e.sel = prev }
}

func peekStrings(src reactive.Readable[[]string]) (v []string) {
	defer func() {
		if recover() != nil {
			v = nil
		}
	}()
	return src.Peek()
}

func optionValue(el *work.Element) string {
	if v, ok := attrState(el, "value"); ok {
		return v
	}
	var text strings.Builder
	for _, c := range el.Children {
		if t, ok := c.(*work.Text); ok {
			text.WriteString(t.Value)
		}
	}
	return text.String()
}

func (e *emitter) markSelected(el *work.Element) {
	if e.sel == nil || el.Tag != "option" {
		return
	}
	if _, ok := e.sel.values[optionValue(el)]; ok {
		el.Attrs = append(el.Attrs, work.Attr{Name: "selected", Kind: work.AttrPresent})
	}
}

func bindFields(names ...string) []string {
	if err := protocol.ValidateFields(names...); err != nil {
		panic(err)
	}
	return names
}

func (s *Session) bindEvents(el *work.Element, spec *work.BindSpec, target bindTarget, id uint32) []work.EventBinding {
	opts := metadata.EventOptions{Debounce: int(spec.Debounce.Milliseconds()), Throttle: spec.Throttle}
	var event string
	var fn func(work.Event)
	switch target {
	case targetChecked:
		event, opts.Fields = metadata.EditEvent, bindFields("checked")
		fn = func(evt work.Event) {
			if v, ok := evt.Payload["checked"].(bool); ok {
				s.noteEcho(id, "checked", strconv.FormatBool(v))
				spec.WriteBool.Set(v)
			}
		}
	case targetOpen:
		event, opts.Fields = "toggle", bindFields("open")
		fn = func(evt work.Event) {
			if v, ok := evt.Payload["open"].(bool); ok {
				s.noteEcho(id, "open", strconv.FormatBool(v))
				spec.WriteBool.Set(v)
			}
		}
	case targetSelection:
		event, opts.Fields = metadata.EditEvent, bindFields("selected")
		fn = func(evt work.Event) {
			vals := protocol.Strings(evt.Payload["selected"])
			s.noteEcho(id, selectionAttr, encodeList(vals))
			spec.WriteStrings.Set(vals)
		}
	default:
		event, opts.Fields = metadata.EditEvent, bindFields("value")
		if spec.Lazy {
			event = "change"
		}
		fn = func(evt work.Event) {
			if v, ok := evt.Payload["value"].(string); ok {
				s.noteEcho(id, "value", v)
				spec.WriteString.Set(v)
			}
		}
	}
	bindings := []work.EventBinding{{Event: event, Options: opts, Fns: []func(work.Event){fn}}}
	if target == targetOpen && el.Tag == "dialog" {
		bindings = append(bindings, work.EventBinding{Event: "close", Options: opts, Fns: []func(work.Event){fn}})
	}
	return bindings
}

func (s *Session) noteEcho(id uint32, name, value string) {
	if s.echo == nil {
		s.echo = make(map[echoKey]string)
	}
	s.echo[echoKey{id, name}] = value
}

func (s *Session) isEcho(id uint32, name, value string) bool {
	if v, ok := s.echo[echoKey{id, name}]; ok && v == value {
		return true
	}
	return false
}
