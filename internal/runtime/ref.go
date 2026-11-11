package runtime

import (
	"fmt"
	"runtime/debug"
	"sort"
	"sync/atomic"

	"github.com/eleven-am/pondlive/internal/effectguard"
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

const (
	WarnRefNotConnected WarningCode = "REF_NOT_CONNECTED"
	WarnRefDetached     WarningCode = "REF_DETACHED"
	WarnRefForbidden    WarningCode = "REF_FORBIDDEN_CALL"
)

type Ref struct {
	attachment
	attached *reactive.Signal[bool]
	attrs    map[string]string
	watches  []*refWatch
	sigs     map[string]any
	seeds    []func(attrs map[string]string)
}

type refWatch struct {
	prop string
	ms   int
}

func NewRef(ctx *Ctx, name string, accepts func(tag string) bool) *Ref {
	r := &Ref{attachment: attachment{s: ctx.s, o: ctx.o, name: name, accepts: accepts}}
	r.attached = reactive.NewSignal(ctx.s.graph, false)
	r.onBind = r.bound
	r.onUnbind = r.unbound
	ctx.o.OnCleanup(r.shutdown)
	return r
}

func UseCell[T any](ctx *Ctx, init func() *T) *T {
	return reactive.Cell(ctx.o, init)
}

func (r *Ref) AttachedSignal() ReadSignal[bool] { return r.attached }

func (r *Ref) bound(el *work.Element) {
	r.attrs = startAttrs(el)
	if r.s.refs == nil {
		r.s.refs = make(map[uint32]*Ref)
	}
	r.s.refs[r.id] = r
	for _, seed := range r.seeds {
		seed(r.attrs)
	}
	r.attached.Set(true)
	r.syncAll()
}

func (r *Ref) unbound() {
	if r.live() {
		for _, w := range r.watches {
			r.s.Bus.PublishDOMUnobserve(protocol.DOMUnobservePayload{ID: r.id, Prop: w.prop})
		}
	}
	if r.s.refs[r.id] == r {
		delete(r.s.refs, r.id)
	}
	r.attrs = nil
	r.attached.Set(false)
}

func (r *Ref) shutdown() {
	if r.id != 0 {
		r.unbound()
		r.id = 0
	}
	r.watches = nil
}

func (r *Ref) live() bool {
	return r.id != 0 && r.s.Bus != nil && !r.s.deferred.Load()
}

func (r *Ref) syncAll() {
	for _, w := range r.watches {
		r.syncWatch(w)
	}
}

func (r *Ref) syncWatch(w *refWatch) {
	if r.live() {
		r.s.Bus.PublishDOMObserve(protocol.DOMObservePayload{K: r.name, ID: r.id, Prop: w.prop, Ms: w.ms})
	}
}

func (s *Session) ResyncRefs() {
	if s == nil || len(s.refs) == 0 {
		return
	}
	ids := make([]uint32, 0, len(s.refs))
	for id := range s.refs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		s.refs[id].syncAll()
	}
}

var refStartAttrs = [...]string{"value", "checked", "muted", "autoplay", "autofocus"}

func startAttrs(el *work.Element) map[string]string {
	var out map[string]string
	put := func(name, v string) {
		if out == nil {
			out = make(map[string]string, 2)
		}
		out[name] = v
	}
	for _, a := range el.Attrs {
		wanted := false
		for _, n := range refStartAttrs {
			if a.Name == n {
				wanted = true
				break
			}
		}
		if !wanted {
			continue
		}
		switch a.Kind {
		case work.AttrStatic:
			put(a.Name, a.Value)
		case work.AttrPresent:
			put(a.Name, "")
		case work.AttrDynamic, work.AttrDynamicOptional:
			put(a.Name, peekString(a.Source))
		case work.AttrDynamicPresence:
			if a.Presence.Peek() {
				put(a.Name, "")
			}
		}
	}
	return out
}

func (r *Ref) here(fn func()) {
	if r.s.graph.OnActor() {
		fn()
		return
	}
	r.s.RunOnActor(fn)
}

func (r *Ref) ready(op string) error {
	if r.s.deferred.Load() {
		r.s.Report(r.o, Report{
			Severity: SeverityWarning,
			Code:     string(WarnRefNotConnected),
			Message:  fmt.Sprintf("%s.%s was called before the browser connected and was dropped; move it to UseEffect or a handler", r.name, op),
			Stack:    string(debug.Stack()),
			Meta:     map[string]any{"call": op},
		})
		return ErrNotConnected
	}
	if r.id == 0 {
		r.s.Report(r.o, Report{
			Severity: SeverityWarning,
			Code:     string(WarnRefDetached),
			Message:  fmt.Sprintf("%s.%s was called on a ref that is not attached to a mounted element and was dropped", r.name, op),
			Stack:    string(debug.Stack()),
			Meta:     map[string]any{"call": op},
		})
		return ErrRefNotAttached
	}
	return nil
}

func (r *Ref) guarded(op string, fn func()) bool {
	if r == nil {
		return false
	}
	if r.s.graph.Mode() == reactive.ModeSerialize {
		return true
	}
	if effectguard.Refuse(r.name + "." + op) {
		return false
	}
	r.here(fn)
	return true
}

func (r *Ref) allow(op protocol.RefOp, names ...string) bool {
	for _, name := range names {
		if !protocol.RefAllows(r.name, op, name) {
			r.s.Report(r.o, Report{
				Severity: SeverityWarning,
				Code:     string(WarnRefForbidden),
				Message:  fmt.Sprintf("%s does not allow %s %q; the call was dropped", r.name, op, name),
			})
			return false
		}
	}
	return true
}

func (r *Ref) Call(method string, args ...any) {
	r.guarded(method, func() {
		if r.allow(protocol.RefMethod, method) && r.ready(method) == nil {
			r.s.Bus.PublishDOMCall(protocol.DOMCallPayload{K: r.name, ID: r.id, Method: method, Args: args})
		}
	})
}

func (r *Ref) Set(prop string, value any) {
	r.guarded("Set "+prop, func() {
		if r.allow(protocol.RefSet, prop) && r.ready("Set "+prop) == nil {
			r.s.Bus.PublishDOMSet(protocol.DOMSetPayload{K: r.name, ID: r.id, Prop: prop, Value: value})
		}
	})
}

func (r *Ref) ask(op string, allowed func() bool, publish func(requestID string, id uint32), fn func(protocol.DOMResponsePayload, error)) {
	if fn == nil {
		return
	}
	if r == nil {
		fn(protocol.DOMResponsePayload{}, ErrNilRef)
		return
	}
	ok := r.guarded(op, func() {
		if !allowed() {
			r.s.RunOnActor(func() { r.s.invokeHandler(r.o, func() { fn(protocol.DOMResponsePayload{}, ErrForbiddenCall) }) })
			return
		}
		if err := r.ready(op); err != nil {
			r.s.RunOnActor(func() { r.s.invokeHandler(r.o, func() { fn(protocol.DOMResponsePayload{}, err) }) })
			return
		}
		id := r.id
		r.s.domAsk(r.o, func(requestID string) { publish(requestID, id) }, fn)
	})
	if !ok {
		fn(protocol.DOMResponsePayload{}, ErrNilRef)
	}
}

func (r *Ref) Query(fn func(map[string]any, error), props ...string) {
	if len(props) == 0 {
		if fn != nil {
			fn(map[string]any{}, nil)
		}
		return
	}
	r.ask("Query", func() bool { return r.allow(protocol.RefQuery, props...) }, func(requestID string, id uint32) {
		r.s.Bus.PublishDOMQuery(protocol.DOMQueryPayload{K: r.name, RequestID: requestID, ID: id, Selectors: props})
	}, func(resp protocol.DOMResponsePayload, err error) {
		fn(resp.Values, err)
	})
}

func (r *Ref) CallThen(fn func(any, error), method string, args ...any) {
	r.ask(method, func() bool { return r.allow(protocol.RefMethod, method) }, func(requestID string, id uint32) {
		r.s.Bus.PublishDOMAsync(protocol.DOMAsyncPayload{K: r.name, RequestID: requestID, ID: id, Method: method, Args: args})
	}, func(resp protocol.DOMResponsePayload, err error) {
		fn(resp.Result, err)
	})
}

type ObserveOpt interface{ ObserveRate() int }

func observeWire(prop string, opts []ObserveOpt) int {
	rate := metadata.RateDefault
	for _, o := range opts {
		rate = o.ObserveRate()
	}
	switch {
	case rate == metadata.RateFrame:
		return protocol.ObserveFrame
	case rate == metadata.RateOff:
		return protocol.ObserveImmediate
	case rate > 0:
		return rate
	case protocol.ObservedPerFrame(prop):
		return protocol.ObserveFrame
	}
	return protocol.ObserveImmediate
}

type RefSpec[T any] struct {
	Prop   string
	Zero   T
	Decode func(any) (T, bool)
	Seed   func(attrs map[string]string) (T, bool)
}

type refSignal[T any] struct {
	r       *Ref
	w       *refWatch
	sig     *reactive.Signal[T]
	decode  func(any) (T, bool)
	started atomic.Bool
}

type refApplier interface{ apply(raw any) }

func (l *refSignal[T]) apply(raw any) {
	if v, ok := l.decode(raw); ok {
		l.sig.Set(v)
	}
}

func (s *Session) applyDOMValue(p protocol.DOMValuePayload) {
	r := s.refs[p.ID]
	if r == nil {
		return
	}
	if l, ok := r.sigs[p.Prop].(refApplier); ok {
		l.apply(p.Value)
	}
}

func (l *refSignal[T]) start() {
	if l.started.CompareAndSwap(false, true) {
		l.r.here(func() {
			l.r.watches = append(l.r.watches, l.w)
			l.r.syncWatch(l.w)
		})
	}
}

func (l *refSignal[T]) Get() T {
	l.start()
	return l.sig.Get()
}

func (l *refSignal[T]) Peek() T {
	l.start()
	return l.sig.Peek()
}

func RefSignal[T any](r *Ref, spec RefSpec[T], opts ...ObserveOpt) ReadSignal[T] {
	if r == nil {
		return Const(spec.Zero)
	}
	if existing, ok := r.sigs[spec.Prop].(*refSignal[T]); ok {
		return existing
	}
	if !r.allow(protocol.RefObserve, spec.Prop) {
		return Const(spec.Zero)
	}
	initial := spec.Zero
	seed := func(attrs map[string]string) (T, bool) {
		if spec.Seed == nil {
			return spec.Zero, false
		}
		return spec.Seed(attrs)
	}
	if v, ok := seed(r.attrs); ok {
		initial = v
	}
	sig := reactive.NewSignal(r.s.graph, initial)
	l := &refSignal[T]{r: r, sig: sig, decode: spec.Decode}
	l.w = &refWatch{prop: spec.Prop, ms: observeWire(spec.Prop, opts)}
	r.seeds = append(r.seeds, func(attrs map[string]string) {
		if v, ok := seed(attrs); ok {
			sig.Set(v)
		}
	})
	if r.sigs == nil {
		r.sigs = make(map[string]any)
	}
	r.sigs[spec.Prop] = l
	return l
}
