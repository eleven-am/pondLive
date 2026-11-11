package runtime

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type scope struct {
	o    *reactive.Owner
	next uint32
	kid  int
	reg  int
}

func (sc *scope) take() uint32 {
	id := sc.next
	sc.next++
	return id
}

type emitter struct {
	s          *Session
	w          *writer
	ser        bool
	rootID     uint32
	rootRegion bool
	rootSeen   bool
	exiting    bool
	sel        *selectState
	tops       []uint32
	lastBound  bool
}

func (s *Session) newEmitter(w *writer, ser bool) *emitter {
	return &emitter{s: s, w: w, ser: ser}
}

func (e *emitter) noteRoot(id uint32, region bool) {
	if !e.rootSeen {
		e.rootSeen = true
		e.rootID = id
		e.rootRegion = region
	}
}

var nameRe = regexp.MustCompile(`^[A-Za-z_:][A-Za-z0-9_:.\-]*$`)

func validName(name string) bool { return nameRe.MatchString(name) }

type InvalidNameError struct {
	Kind string
	Name string
}

func (e *InvalidNameError) Error() string {
	return fmt.Sprintf("runtime: invalid %s name %q", e.Kind, e.Name)
}

type ReservedNameError struct {
	Name string
}

func (e *ReservedNameError) Error() string {
	return fmt.Sprintf("runtime: attribute %q is reserved by the framework", e.Name)
}

type RoutedAttrError struct {
	Name, Use string
}

func (e *RoutedAttrError) Error() string {
	return fmt.Sprintf("runtime: attribute %q is set with %s, not Attr", e.Name, e.Use)
}

var routedAttrs = map[string]string{"class": "Class", "style": "Style"}

func (e *emitter) node(sc *scope, n work.Node, force bool) {
	switch x := n.(type) {
	case nil:
	case *work.Element:
		e.element(sc, x, force)
	case *work.Text:
		e.staticText(x.Value)
	case *work.BoundText:
		e.markerText(sc, x)
	case *work.Comment:
		e.lastBound = false
		e.w.nl = false
		e.w.raw("<!--")
		e.w.raw(sanitizeComment(x.Value))
		e.w.raw("-->")
	case *work.Fragment:
		e.children(sc, x.Children)
	case *work.ComponentNode:
		e.component(sc, x, force)
	case *work.RegionNode:
		e.region(sc, x, force)
	case *work.UnitRegion:
		e.unitRegion(sc, x)
	case *work.PortalNode:
		e.portal(sc, x)
	case *work.PortalTarget:
		e.portalTarget(sc, force)
	}
}

func (e *emitter) unitRegion(sc *scope, u *work.UnitRegion) {
	id := sc.take()
	e.noteRoot(id, true)
	e.lastBound = false
	e.w.regionStart(id)
	exitAll := e.exiting
	e.exiting = false
	forEachChild(u.Children, func(c work.Node) {
		top := isElement(c)
		e.exiting = exitAll && top
		if top {
			e.tops = append(e.tops, sc.next)
		}
		e.node(sc, c, top)
		e.exiting = false
	})
	e.lastBound = false
	e.w.regionEnd(id)
}

func (e *emitter) children(sc *scope, children []work.Node) {
	forEachChild(children, func(c work.Node) { e.node(sc, c, false) })
}

func (e *emitter) staticText(v string) {
	if v == "" {
		return
	}
	if e.lastBound {
		e.w.raw("<!---->")
		e.lastBound = false
	}
	e.w.text(e.w.parentTag(), v)
}

func (e *emitter) value(sc *scope, kind bindKind, id uint32, name string, src reactive.Readable[string]) string {
	if e.ser {
		return peekString(src)
	}
	b := e.s.startBinding(sc.o, &binding{kind: kind, id: id, name: name, src: src})
	v := b.init
	b.init = ""
	return v
}

func (e *emitter) markerText(sc *scope, bt *work.BoundText) {
	id := sc.take()
	e.w.textMarker(id)
	v := e.value(sc, bindText, id, "", bt.Source)
	e.w.text(e.w.parentTag(), v)
	e.lastBound = true
}

func (e *emitter) element(sc *scope, el *work.Element, force bool) {
	s := e.s
	if !validName(el.Tag) {
		panic(&InvalidNameError{Kind: "tag", Name: el.Tag})
	}
	sole := soleBoundText(el)
	dyn := force || el.Dynamic() || sole != nil
	var id uint32
	if dyn {
		id = sc.take()
		if force {
			e.noteRoot(id, false)
		}
	}
	e.lastBound = false
	w := e.w
	w.openTag(el.Tag)
	if dyn {
		w.attrUint(protocol.AttrID, id)
	}
	e.checkBindOwned(sc, el)
	var bodyText *string
	var leaveSelect func()
	if el.Bind != nil {
		bodyText, leaveSelect = e.applyBind(sc, el, id)
	}
	e.markSelected(el)
	events := s.collectEvents(el)
	if len(events) > 0 {
		e.checkRates(sc, el, events)
		w.attr(protocol.AttrHandlers, string(appendSpecs(make([]byte, 0, 96), events)))
		if !e.ser {
			s.registerHandlers(sc.o, id, events)
		}
	}
	writeAttachmentAttrs(w, el)
	if el.Raw != nil {
		w.raw(" " + protocol.AttrRaw + `=""`)
	}
	e.attrs(sc, el, id)
	if force && e.exiting {
		w.raw(" " + protocol.AttrExiting + `="" inert=""`)
		e.exiting = false
	}
	if !e.ser && len(el.Attachments) > 0 {
		s.bindAttachments(sc.o, el, id)
	}
	w.closeOpen()
	if isVoid(el.Tag) {
		w.leave()
		return
	}
	w.nl = el.Tag == "textarea" || el.Tag == "pre" || el.Tag == "listing"
	switch {
	case el.Raw != nil:
		e.rawBody(sc, el, id)
	case sole != nil:
		v := e.value(sc, bindText, id, "", sole.Source)
		w.text(el.Tag, v)
	case bodyText != nil:
		w.text(el.Tag, *bodyText)
	default:
		e.children(sc, el.Children)
	}
	e.lastBound = false
	w.closeTag(el.Tag)
	if leaveSelect != nil {
		leaveSelect()
	}
}

func (e *emitter) rawBody(sc *scope, el *work.Element, id uint32) {
	if el.Raw.Source == nil {
		e.w.raw(normalizeRaw(el.Tag, el.Raw.Static))
		return
	}
	if e.ser {
		e.w.raw(normalizeRaw(el.Tag, peekString(el.Raw.Source)))
		return
	}
	b := e.s.startBinding(sc.o, &binding{kind: bindRaw, id: id, src: el.Raw.Source, tag: el.Tag})
	e.w.raw(b.init)
	b.init = ""
}

func (e *emitter) attrs(sc *scope, el *work.Element, id uint32) {
	w := e.w
	for _, a := range el.Attrs {
		if !e.attrAllowed(a.Name) {
			continue
		}
		switch a.Kind {
		case work.AttrStatic:
			w.attr(a.Name, a.Value)
		case work.AttrPresent:
			w.attr(a.Name, "")
		case work.AttrDynamic:
			w.attr(a.Name, e.value(sc, bindAttr, id, a.Name, a.Source))
		case work.AttrDynamicOptional:
			if v := e.value(sc, bindAttrOptional, id, a.Name, a.Source); v != "" {
				w.attr(a.Name, v)
			}
		case work.AttrDynamicPresence:
			if e.presence(sc, id, a.Name, a.Presence) {
				w.attr(a.Name, "")
			}
		}
	}
	if classParts := el.Classes; len(classParts) > 0 {
		dynamic := false
		for _, p := range classParts {
			dynamic = dynamic || p.Dynamic()
		}
		var v string
		if dynamic && !e.ser {
			b := e.s.startBinding(sc.o, &binding{kind: bindClass, id: id, name: "class", parts: classParts})
			v = b.init
			b.init = ""
		} else {
			v = joinClass(classParts, false)
		}
		if v != "" {
			w.attr("class", v)
		}
	}
	if len(el.Styles) > 0 {
		e.style(sc, el, id)
	}
}

func (e *emitter) attrAllowed(name string) bool {
	if !validName(name) {
		if e.s.devMode {
			panic(&InvalidNameError{Kind: "attribute", Name: name})
		}
		e.s.reportName("attribute:name", name)
		return false
	}
	if protocol.IsReservedAttr(name) {
		if e.s.devMode {
			panic(&ReservedNameError{Name: name})
		}
		e.s.reportName("attribute:reserved", name)
		return false
	}
	if use, ok := routedAttrs[strings.ToLower(name)]; ok {
		if e.s.devMode {
			panic(&RoutedAttrError{Name: name, Use: use})
		}
		e.s.reportName("attribute:routed", name)
		return false
	}
	return true
}

func (s *Session) reportName(phase, name string) {
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: phase,
		Message: fmt.Sprintf("attribute %q dropped", name),
		Meta:    map[string]any{"attribute": name},
	})
}

func (e *emitter) presence(sc *scope, id uint32, name string, src reactive.Readable[bool]) bool {
	if e.ser {
		return peekBool(src)
	}
	b := e.s.startBinding(sc.o, &binding{kind: bindPresence, id: id, name: name, pres: src})
	return b.on
}

func (e *emitter) style(sc *scope, el *work.Element, id uint32) {
	var sb strings.Builder
	dynamic := false
	for _, d := range el.Styles {
		v := d.Value
		if d.Source != nil {
			dynamic = true
			v = e.value(sc, bindStyle, id, d.Prop, d.Source)
		}
		if v == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(d.Prop)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteByte(';')
	}
	if sb.Len() > 0 || dynamic {
		e.w.attr("style", sb.String())
	}
}

type handlerSlot struct {
	event string
	opts  metadata.EventOptions
	fns   []func(work.Event)
}

type handlerFn struct {
	event string
	fn    func(work.Event)
	more  []func(work.Event)
}

func (s *Session) collectEvents(el *work.Element) []handlerSlot {
	if len(el.Events) == 0 {
		return nil
	}
	if len(el.Events) == 1 {
		b := el.Events[0]
		return []handlerSlot{{event: b.Event, opts: b.Options, fns: b.Fns}}
	}
	var out []handlerSlot
	add := func(event string, opts metadata.EventOptions, fns ...func(work.Event)) {
		for i := range out {
			if out[i].event == event {
				out[i].opts = metadata.MergeHandlers(out[i].opts, opts)
				out[i].fns = append(out[i].fns, fns...)
				return
			}
		}
		out = append(out, handlerSlot{event: event, opts: opts, fns: append([]func(work.Event){}, fns...)})
	}
	for _, b := range el.Events {
		add(b.Event, b.Options, b.Fns...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].event < out[j].event })
	return out
}

const (
	WarnEventRateBoth     WarningCode = "EVENT_RATE_BOTH"
	WarnEventRateConflict WarningCode = "EVENT_RATE_CONFLICT"
)

func (e *emitter) checkRates(sc *scope, el *work.Element, events []handlerSlot) {
	if e.ser {
		return
	}
	for _, ev := range events {
		code, msg := WarningCode(""), ""
		switch {
		case ev.opts.Debounce > 0 && ev.opts.Throttle > 0:
			code, msg = WarnEventRateBoth, "both Debounce and Throttle were set; Debounce is used"
		case ev.opts.RateMixed:
			code, msg = WarnEventRateConflict, "handlers on one element ask for different rates for the same event; the finest rate is used for all of them"
		}
		if code != "" {
			e.s.Report(sc.o, Report{Severity: SeverityWarning, Code: string(code), Message: msg, Meta: map[string]any{"tag": el.Tag, "event": ev.event}})
		}
	}
}

func appendSpecs(dst []byte, events []handlerSlot) []byte {
	dst = append(dst, '[')
	for i, ev := range events {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendSpecKey(dst, protocol.SpecEvent, false)
		dst = protocol.AppendJSONString(dst, ev.event)
		o := ev.opts
		r := metadata.Resolve(ev.event, o)
		for _, flag := range []struct {
			on  bool
			key string
		}{{o.Prevent, protocol.SpecPrevent}, {o.Stop, protocol.SpecStop}, {r.Passive, protocol.SpecPassive}, {o.Once, protocol.SpecOnce}, {o.Capture, protocol.SpecCapture}, {o.CapturePointer, protocol.SpecCapturePointer}} {
			if flag.on {
				dst = appendSpecKey(dst, flag.key, true)
				dst = append(dst, '1')
			}
		}
		if r.Debounce > 0 {
			dst = appendSpecKey(dst, protocol.SpecDebounce, true)
			dst = strconv.AppendInt(dst, int64(r.Debounce), 10)
		}
		if r.Rate {
			dst = appendSpecKey(dst, protocol.SpecThrottle, true)
			dst = strconv.AppendInt(dst, int64(r.Throttle), 10)
		}
		if len(o.Keys) > 0 {
			dst = appendSpecKey(dst, protocol.SpecKeys, true)
			dst = append(dst, '[')
			for j, k := range o.Keys {
				if j > 0 {
					dst = append(dst, ',')
				}
				dst = protocol.AppendJSONString(dst, k)
			}
			dst = append(dst, ']')
		}
		if len(o.Fields) > 0 && o.Fields[0] != metadata.FieldsAll {
			dst = appendSpecKey(dst, protocol.SpecFields, true)
			dst = protocol.AppendJSONString(dst, strings.Join(o.Fields, " "))
		}
		dst = append(dst, '}')
	}
	return append(dst, ']')
}

func appendSpecKey(dst []byte, key string, comma bool) []byte {
	if comma {
		dst = append(dst, ',')
	} else {
		dst = append(dst, '{')
	}
	dst = protocol.AppendJSONString(dst, key)
	return append(dst, ':')
}

func scriptSpecJSON(m *metadata.ScriptMeta) string {
	dst := appendSpecKey(nil, protocol.ScriptSpecID, false)
	dst = protocol.AppendJSONString(dst, m.ScriptID)
	dst = appendSpecKey(dst, protocol.ScriptSpecSource, true)
	dst = protocol.AppendJSONString(dst, m.Script)
	return string(append(dst, '}'))
}

func jsonString(v string) string { return string(protocol.AppendJSONString(nil, v)) }
