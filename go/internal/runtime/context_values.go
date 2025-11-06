package runtime

import (
	"fmt"
	"sync/atomic"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type contextID uint64

type contextCounter struct{ atomic.Uint64 }

var globalContextCounter contextCounter

// Context represents a typed value shared through the component tree.
type Context[T any] struct {
	id  contextID
	def T
	eq  func(a, b T) bool
}

// NewContext constructs a new typed context with the provided default value.
func NewContext[T any](def T) Context[T] {
	id := contextID(globalContextCounter.Add(1))
	return Context[T]{id: id, def: def}
}

// WithEqual configures a custom equality comparer for the context provider state.
func (c Context[T]) WithEqual(eq func(a, b T) bool) Context[T] {
	c.eq = eq
	return c
}

// Provide makes value available to all descendants rendered within the current component.
func (c Context[T]) Provide(ctx Ctx, value T, render func() h.Node) h.Node {
	if render == nil {
		return h.Fragment()
	}
	if ctx.comp == nil {
		return render()
	}
	entry := ensureProviderEntry(ctx, c, value, true)
	entry.owner = ctx.comp
	entry.active = true
	entry.derived = false
	entry.hasLast = false
	if entry.assign != nil {
		if entry.eq == nil || !entry.eq(entry.get(), value) {
			entry.assign(value)
		}
	}
	return render()
}

// ProvideFunc computes the value using compute and provides it to descendants.
func (c Context[T]) ProvideFunc(ctx Ctx, compute func() T, deps []any, render func() h.Node) h.Node {
	if render == nil {
		return h.Fragment()
	}
	if compute == nil {
		var zero T
		return c.Provide(ctx, zero, render)
	}
	value := UseMemo(ctx, compute, deps...)
	if ctx.comp == nil {
		return render()
	}
	entry := ensureProviderEntry(ctx, c, value, true)
	entry.owner = ctx.comp
	entry.active = true
	entry.derived = true
	if entry.assign != nil {
		if !entry.hasLast || entry.eq == nil || !entry.eq(entry.last, value) {
			entry.assign(value)
			entry.last = value
			entry.hasLast = true
		}
	} else {
		entry.last = value
		entry.hasLast = true
	}
	return render()
}

// Use returns the nearest context value or the default when no provider exists.
func (c Context[T]) Use(ctx Ctx) T {
	if ctx.comp == nil {
		return c.def
	}
	if entry := findProviderEntry(ctx.comp, c); entry != nil {
		return entry.get()
	}
	return c.def
}

// UsePair returns the nearest context value and a setter to update the provider.
func (c Context[T]) UsePair(ctx Ctx) (func() T, func(T)) {
	if ctx.comp == nil {
		return func() T { return c.def }, func(T) {}
	}
	entry := findProviderEntry(ctx.comp, c)
	isOwner := entry != nil && entry.owner == ctx.comp
	local := ensureProviderEntry(ctx, c, c.def, false)
	if isOwner {
		return entry.get, entry.set
	}
	if local.eq == nil {
		equal := c.eq
		if equal == nil {
			equal = defaultEqual[T]()
		}
		local.eq = equal
	}
	if entry != nil {
		getter := func() T {
			if local.active {
				return local.get()
			}
			return entry.get()
		}
		setter := func(v T) {
			if local.active {
				local.set(v)
				return
			}
			if local.assign != nil {
				local.assign(v)
			}
			entry.set(v)
		}
		return getter, setter
	}
	return local.get, local.set
}

// UseSelect derives a sub-value from the context and only re-renders when it changes.
func UseSelect[T any, U any](ctx Ctx, c Context[T], pick func(T) U, eq func(U, U) bool) U {
	if pick == nil {
		var zero U
		return zero
	}
	whole := c.Use(ctx)
	selected := pick(whole)
	comparer := eq
	if comparer == nil {
		comparer = defaultEqual[U]()
	}
	type selectState struct {
		ok  bool
		val U
	}
	ref := UseRef(ctx, selectState{})
	if !ref.Cur.ok || comparer == nil || !comparer(ref.Cur.val, selected) {
		ref.Cur = selectState{ok: true, val: selected}
	}
	return ref.Cur.val
}

// Require returns the nearest context value or panics when none exists.
func (c Context[T]) Require(ctx Ctx) T {
	if ctx.comp == nil {
		panic("runtime: context Require called outside component")
	}
	entry := findProviderEntry(ctx.comp, c)
	if entry == nil {
		compName := "<component>"
		if ctx.comp.callable != nil {
			compName = ctx.comp.callable.name()
		}
		panic(fmt.Sprintf("runtime: missing provider for context %d in %s", c.id, compName))
	}
	return entry.get()
}

type providerEntry[T any] struct {
	get     func() T
	set     func(T)
	assign  func(T)
	eq      func(a, b T) bool
	owner   *component
	active  bool
	derived bool
	hasLast bool
	last    T
}

func ensureProviderEntry[T any](ctx Ctx, c Context[T], initial T, activate bool) *providerEntry[T] {
	if ctx.comp == nil {
		eq := c.eq
		if eq == nil {
			eq = defaultEqual[T]()
		}
		return &providerEntry[T]{
			get: func() T { return initial },
			set: func(T) {},
			assign: func(v T) {
				initial = v
			},
			eq:     eq,
			active: activate,
		}
	}
	if ctx.comp.providers == nil {
		ctx.comp.providers = make(map[contextID]any)
	}
	existing, providerExists := ctx.comp.providers[c.id]
	comparer := c.eq
	if comparer == nil {
		comparer = defaultEqual[T]()
	}
	baseIdx := ctx.frame.idx
	get, baseSet := UseState(ctx, initial, WithEqual(comparer))
	if providerExists {
		entry, ok := existing.(*providerEntry[T])
		if !ok {
			panic(fmt.Sprintf("runtime: context type mismatch for %d", c.id))
		}
		if activate {
			entry.active = true
		}
		return entry
	}
	var cellPtr *stateCell[T]
	if baseIdx < len(ctx.frame.cells) {
		if cell, ok := ctx.frame.cells[baseIdx].(*stateCell[T]); ok {
			cellPtr = cell
		}
	}
	owner := ctx.comp
	mark := func(prev, next T) {
		if owner == nil {
			return
		}
		if comparer != nil && comparer(prev, next) {
			return
		}
		if atomic.LoadInt32(&owner.rendering) == 1 {
			owner.markDescendantsDirtyLocked()
			return
		}
		owner.markDescendantsDirty()
	}
	set := func(v T) {
		prev := get()
		baseSet(v)
		mark(prev, v)
	}
	entry := &providerEntry[T]{
		get: get,
		set: set,
		assign: func(v T) {
			if cellPtr != nil {
				prev := cellPtr.val
				cellPtr.val = v
				mark(prev, v)
				return
			}
			set(v)
		},
		eq:     comparer,
		owner:  ctx.comp,
		active: activate,
	}
	ctx.comp.providers[c.id] = entry
	return entry
}

func findProviderEntry[T any](comp *component, c Context[T]) *providerEntry[T] {
	for current := comp; current != nil; current = current.parent {
		if current.providers == nil {
			continue
		}
		if raw, ok := current.providers[c.id]; ok {
			entry, ok := raw.(*providerEntry[T])
			if !ok {
				panic(fmt.Sprintf("runtime: context type mismatch for %d", c.id))
			}
			if entry.active {
				return entry
			}
		}
	}
	return nil
}
