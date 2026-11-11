package runtime

import (
	"reflect"
	"sync/atomic"

	"github.com/eleven-am/pondlive/internal/reactive"
)

type Signal[T any] = reactive.Signal[T]

type SignalOpt[T any] = reactive.Option[T]

func WithEqual[T any](eq func(a, b T) bool) SignalOpt[T] { return reactive.WithEqual(eq) }

func UseSignal[T any](ctx *Ctx, initial T, opts ...SignalOpt[T]) *Signal[T] {
	return reactive.Cell(ctx.o, func() *Signal[T] {
		return reactive.NewSignal(ctx.s.graph, initial, opts...)
	})
}

func UseComputed[T any](ctx *Ctx, fn func() T, opts ...SignalOpt[T]) ReadSignal[T] {
	return reactive.Cell(ctx.o, func() *reactive.Computed[T] {
		var last T
		s, o := ctx.s, ctx.o
		guarded := func() (v T) {
			defer func() {
				if r := recover(); r != nil {
					s.computedPanic(o, r)
					v = last
				}
			}()
			v = fn()
			last = v
			return v
		}
		return reactive.NewComputed(ctx.o, guarded, opts...)
	})
}

func (s *Session) computedPanic(o *reactive.Owner, r any) {
	if _, violation := r.(*reactive.DevError); violation && s.devMode {
		panic(r)
	}
	s.panicToBoundary(o, r, true)
}

type constSignal[T any] struct{ v T }

func (c constSignal[T]) Get() T  { return c.v }
func (c constSignal[T]) Peek() T { return c.v }

func Const[T any](v T) ReadSignal[T] { return constSignal[T]{v: v} }

type derived[A, B any] struct {
	src ReadSignal[A]
	fn  func(A) B
}

func (d derived[A, B]) Get() B  { return d.fn(d.src.Get()) }
func (d derived[A, B]) Peek() B { return d.fn(d.src.Peek()) }

func Derive[A, B any](src ReadSignal[A], fn func(A) B) ReadSignal[B] {
	return derived[A, B]{src: src, fn: fn}
}

type watchCell struct {
	s       *Session
	o       *reactive.Owner
	fn      func() func()
	gate    *reactive.Signal[bool]
	eff     *reactive.Effect
	cleanup func()
}

var effectRuns atomic.Uint64

func EffectRuns() uint64 { return effectRuns.Load() }

func (w *watchCell) run() {
	if !w.gate.Get() {
		return
	}
	w.runCleanup()
	effectRuns.Add(1)
	err := w.s.recoverUser(ErrCodeEffect, "effect:run", w.o, func() { w.cleanup = w.fn() })
	if err != nil {
		w.s.deliver(w.o, err)
	}
}

func (w *watchCell) runCleanup() {
	if c := w.cleanup; c != nil {
		w.cleanup = nil
		if err := w.s.recoverUser(ErrCodeEffectCleanup, "effect:cleanup", w.o, c); err != nil {
			w.s.deliver(w.o, err)
		}
	}
}

func UseEffect(ctx *Ctx, fn func() func()) {
	reactive.Cell(ctx.o, func() *watchCell {
		s := ctx.s
		w := &watchCell{s: s, o: ctx.o, fn: fn, gate: reactive.NewSignal(s.graph, false)}
		w.eff = reactive.NewEffect(ctx.o, reactive.KindWatch, w.run)
		ctx.o.OnCleanup(w.runCleanup)
		s.arming = append(s.arming, func() { w.gate.Set(true) })
		return w
	})
}

func (s *Session) runArmed() {
	if s.armRunning {
		return
	}
	s.armRunning = true
	defer func() { s.armRunning = false }()
	for len(s.arming) > 0 {
		fn := s.arming[0]
		s.arming = s.arming[1:]
		fn()
	}
	s.arming = nil
}

type Context[T any] struct {
	name string
}

func NewContext[T any]() *Context[T] {
	return &Context[T]{name: reflect.TypeFor[T]().String()}
}

func Provide[T any](ctx *Ctx, c *Context[T], sig *Signal[T]) {
	if ctx.s.graph.Mode() == reactive.ModeSerialize {
		return
	}
	ctx.o.SetContext(c, sig)
}

type MissingProviderError struct {
	Type string
}

func (e *MissingProviderError) Error() string {
	return "context " + e.Type + " has no provider"
}

type contextCell[T any] struct {
	sig *Signal[T]
}

func UseContext[T any](ctx *Ctx, c *Context[T]) *Signal[T] {
	cell := reactive.Cell(ctx.o, func() *contextCell[T] {
		v, ok := ctx.o.Lookup(c)
		if !ok {
			panic(&MissingProviderError{Type: c.name})
		}
		return &contextCell[T]{sig: v.(*Signal[T])}
	})
	return cell.sig
}
