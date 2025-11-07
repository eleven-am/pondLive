package runtime

import (
	"reflect"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type StateOpt[T any] interface{ applyStateOpt(*stateCell[T]) }

type stateOptFunc[T any] func(*stateCell[T])

func (f stateOptFunc[T]) applyStateOpt(c *stateCell[T]) { f(c) }

// WithEqual overrides the equality comparer for a state cell.
func WithEqual[T any](eq func(a, b T) bool) StateOpt[T] {
	return stateOptFunc[T](func(cell *stateCell[T]) {
		if eq == nil {
			cell.eq = defaultEqual[T]()
			return
		}
		cell.eq = eq
	})
}

type stateCell[T any] struct {
	val   T
	eq    func(a, b T) bool
	owner *component
}

func defaultEqual[T any]() func(a, b T) bool {
	return func(a, b T) bool {
		va := reflect.ValueOf(a)
		vb := reflect.ValueOf(b)
		if !va.IsValid() && !vb.IsValid() {
			return true
		}
		if va.Type() == vb.Type() && va.Type().Comparable() {
			return va.Interface() == vb.Interface()
		}
		return reflect.DeepEqual(a, b)
	}
}

// UseState provides component-local state with equality checks to avoid redundant renders.
func UseState[T any](ctx Ctx, initial T, opts ...StateOpt[T]) (func() T, func(T)) {
	if ctx.frame == nil {
		panic("runtime: UseState called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++
	if idx >= len(ctx.frame.cells) {
		cell := &stateCell[T]{
			val:   initial,
			eq:    defaultEqual[T](),
			owner: ctx.comp,
		}
		for _, opt := range opts {
			if opt != nil {
				opt.applyStateOpt(cell)
			}
		}
		ctx.frame.cells = append(ctx.frame.cells, cell)
	}
	cell, ok := ctx.frame.cells[idx].(*stateCell[T])
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseState", ctx.frame.cells[idx])
	}
	get := func() T { return cell.val }
	set := func(next T) {
		if cell.eq != nil && cell.eq(cell.val, next) {
			return
		}
		cell.val = next
		if ctx.sess != nil {
			ctx.sess.markDirty(cell.owner)
		}
	}
	return get, set
}

type Ref[T any] struct {
	Cur T
}

// UseRef returns a stable mutable reference that does not trigger renders when mutated.
func UseRef[T any](ctx Ctx, zero T) *Ref[T] {
	if ctx.frame == nil {
		panic("runtime: UseRef called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++
	if idx >= len(ctx.frame.cells) {
		ctx.frame.cells = append(ctx.frame.cells, &Ref[T]{Cur: zero})
	}
	ref, ok := ctx.frame.cells[idx].(*Ref[T])
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseRef", ctx.frame.cells[idx])
	}
	return ref
}

type elementRefCell[T h.ElementDescriptor] struct {
	ref   *h.ElementRef[T]
	state any
}

func (c *elementRefCell[T]) resetAttachment() {
	if c == nil || c.ref == nil {
		return
	}
	c.ref.ResetAttachment()
}

// UseElement returns a typed ElementRef that can be attached to a generated
// element. The ref caches state via a UseState cell and carries a stable ref ID
// for serialization.
func UseElement[T h.ElementDescriptor](ctx Ctx) *h.ElementRef[T] {
	if ctx.frame == nil {
		panic("runtime: UseElement called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++
	if idx >= len(ctx.frame.cells) {
		if ctx.sess == nil {
			panic("runtime: UseElement requires an active session")
		}
		var descriptor T
		id := ctx.sess.allocateElementRefID()
		ref := h.NewElementRef[T](id, descriptor)
		cell := &elementRefCell[T]{ref: ref}
		ref.InstallState(
			func() any {
				return cell.state
			},
			func(next any) {
				if reflect.DeepEqual(cell.state, next) {
					return
				}
				cell.state = next
				if ctx.sess != nil {
					ctx.sess.markDirty(ctx.comp)
				}
			},
		)
		h.ApplyRefDefaults(ref)
		ctx.sess.registerElementRef(ref)
		ctx.frame.cells = append(ctx.frame.cells, cell)
	}
	cell, ok := ctx.frame.cells[idx].(*elementRefCell[T])
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseElement", ctx.frame.cells[idx])
	}
	return cell.ref
}

type memoCell[T any] struct {
	val  T
	deps []any
}

// UseMemo recomputes a value only when dependencies change.
func UseMemo[T any](ctx Ctx, compute func() T, deps ...any) T {
	if ctx.frame == nil {
		panic("runtime: UseMemo called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++
	if idx >= len(ctx.frame.cells) {
		val := compute()
		ctx.frame.cells = append(ctx.frame.cells, &memoCell[T]{val: val, deps: cloneDeps(deps)})
		return val
	}
	cell, ok := ctx.frame.cells[idx].(*memoCell[T])
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseMemo", ctx.frame.cells[idx])
	}
	if !depsEqual(cell.deps, deps) {
		cell.val = compute()
		cell.deps = cloneDeps(deps)
	}
	return cell.val
}

type Cleanup func()

type effectCell struct {
	deps    []any
	cleanup Cleanup
}

// UseEffect schedules side effects after the next flush and handles cleanup when deps change or unmount.
func UseEffect(ctx Ctx, setup func() Cleanup, deps ...any) {
	if ctx.frame == nil {
		panic("runtime: UseEffect called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++
	if idx >= len(ctx.frame.cells) {
		cell := &effectCell{deps: cloneDeps(deps)}
		ctx.frame.cells = append(ctx.frame.cells, cell)
		if ctx.sess != nil {
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
		return
	}
	cell, ok := ctx.frame.cells[idx].(*effectCell)
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseEffect", ctx.frame.cells[idx])
	}
	if !depsEqual(cell.deps, deps) {
		if ctx.sess != nil {
			ctx.sess.enqueueCleanup(ctx.comp, idx)
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
		cell.deps = cloneDeps(deps)
	} else if cell.cleanup == nil {

		if ctx.sess != nil {
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
	}
}

func cloneDeps(deps []any) []any {
	if len(deps) == 0 {
		return nil
	}
	out := make([]any, len(deps))
	copy(out, deps)
	return out
}

func depsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
