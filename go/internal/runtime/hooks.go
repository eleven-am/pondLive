package runtime

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/eleven-am/pondlive/go/internal/work"
)

// Ref represents a mutable reference that persists across renders.
type Ref[T any] struct {
	Current T
}

// StateOpt configures a state cell.
type StateOpt[T any] interface{ applyStateOpt(*stateCell[T]) }

type stateOptFunc[T any] func(*stateCell[T])

func (f stateOptFunc[T]) applyStateOpt(c *stateCell[T]) { f(c) }

// WithEqual overrides the equality comparer for a state cell.
// When the equality function returns true, the setter will skip marking the component dirty.
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
	owner *Instance
}

// defaultEqual returns a default equality function using reflect.DeepEqual.
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
// Returns the current value and a setter function.
// Options can be passed to customize behavior (e.g., WithEqual for custom equality).
func UseState[T any](ctx *Ctx, initial T, opts ...StateOpt[T]) (T, func(T)) {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		cell := &stateCell[T]{
			val:   initial,
			eq:    defaultEqual[T](),
			owner: ctx.instance,
		}
		for _, opt := range opts {
			if opt != nil {
				opt.applyStateOpt(cell)
			}
		}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeState,
			Value: cell,
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*stateCell[T])
	if !ok {
		panic("runtime: UseState hook mismatch")
	}

	set := func(next T) {
		old := cell.val

		if cell.eq != nil && cell.eq(old, next) {
			return
		}
		cell.val = next

		if ctx.session != nil {
			ctx.session.MarkDirty(cell.owner)
		}
	}

	return cell.val, set
}

// UseRef returns a stable mutable reference that does not trigger renders when mutated.
func UseRef[T any](ctx *Ctx, initial T) *Ref[T] {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeRef,
			Value: &Ref[T]{Current: initial},
		})
	}

	ref, ok := ctx.instance.HookFrame[idx].Value.(*Ref[T])
	if !ok {
		panic("runtime: UseRef hook mismatch")
	}

	return ref
}

// UseElement creates and returns a stable element reference.
// The ref persists across renders and provides a stable ID for element attachment.
//
// Use this base hook in runtime. The html package will provide typed wrappers
// like UseButton, UseDiv, etc. that embed this and add APIs.
func UseElement(ctx *Ctx) *ElementRef {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		id := fmt.Sprintf("%s:r%d", ctx.instance.ID, idx)
		ref := &ElementRef{
			id:       id,
			handlers: make(map[string][]work.Handler),
		}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeElement,
			Value: ref,
		})
	}

	ref, ok := ctx.instance.HookFrame[idx].Value.(*ElementRef)
	if !ok {
		panic("runtime: UseElement hook mismatch")
	}

	ref.ResetAttachment()

	return ref
}

type memoCell[T any] struct {
	val  T
	deps []any
}

// UseMemo recomputes a value only when dependencies change.
// Dependencies are compared using smart equality that handles functions, channels, and maps.
func UseMemo[T any](ctx *Ctx, compute func() T, deps ...any) T {
	idx := ctx.hookIndex
	ctx.hookIndex++

	safeCompute := func() (result T, err *ComponentError) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				err = &ComponentError{
					Message:     fmt.Sprintf("%v", r),
					StackTrace:  stack,
					ComponentID: ctx.instance.ID,
					Phase:       "memo",
					HookIndex:   idx,
					Timestamp:   time.Now(),
				}

				if ctx.session != nil && ctx.session.devMode && ctx.session.reporter != nil {
					ctx.session.reporter.ReportDiagnostic(Diagnostic{
						Phase:      fmt.Sprintf("memo:%s:%d", ctx.instance.ID, idx),
						Message:    fmt.Sprintf("panic: %v", r),
						StackTrace: stack,
						Metadata: map[string]any{
							"component_id": ctx.instance.ID,
							"hook_index":   idx,
							"panic_value":  r,
						},
					})
				}
			}
		}()
		result = compute()
		return result, nil
	}

	if idx >= len(ctx.instance.HookFrame) {
		val, err := safeCompute()
		if err != nil {

			ctx.instance.mu.Lock()
			ctx.instance.RenderError = err
			ctx.instance.mu.Unlock()
		}
		cell := &memoCell[T]{val: val, deps: cloneDeps(deps)}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeMemo,
			Value: cell,
		})
		return val
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*memoCell[T])
	if !ok {
		panic("runtime: UseMemo hook mismatch")
	}

	if !depsEqual(cell.deps, deps) {
		val, err := safeCompute()
		if err != nil {

			ctx.instance.mu.Lock()
			ctx.instance.RenderError = err
			ctx.instance.mu.Unlock()
			return cell.val
		}
		cell.val = val
		cell.deps = cloneDeps(deps)
	}

	return cell.val
}

type effectCell struct {
	cleanup func()
	deps    []any
}

// UseEffect runs side effects with optional cleanup.
// Dependencies are compared using smart equality that handles functions, channels, and maps.
// Effects are deferred and run after the flush completes.
func UseEffect(ctx *Ctx, fn func() func(), deps ...any) {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		cell := &effectCell{cleanup: nil, deps: cloneDeps(deps)}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeEffect,
			Value: cell,
		})

		if ctx.session != nil {
			ctx.session.PendingEffects = append(ctx.session.PendingEffects, effectTask{
				instance:  ctx.instance,
				hookIndex: idx,
				fn:        fn,
			})
		}
		return
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*effectCell)
	if !ok {
		panic("runtime: UseEffect hook mismatch")
	}

	if len(deps) == 0 {

		if cell.cleanup != nil && ctx.session != nil {
			cleanup := cell.cleanup
			ctx.session.PendingCleanups = append(ctx.session.PendingCleanups, cleanupTask{
				instance: ctx.instance,
				fn:       cleanup,
			})
			cell.cleanup = nil
		}

		if ctx.session != nil {
			ctx.session.PendingEffects = append(ctx.session.PendingEffects, effectTask{
				instance:  ctx.instance,
				hookIndex: idx,
				fn:        fn,
			})
		}
		return
	}

	if !depsEqual(cell.deps, deps) {
		if cell.cleanup != nil && ctx.session != nil {
			cleanup := cell.cleanup
			ctx.session.PendingCleanups = append(ctx.session.PendingCleanups, cleanupTask{
				instance: ctx.instance,
				fn:       cleanup,
			})
			cell.cleanup = nil
		}

		if ctx.session != nil {
			ctx.session.PendingEffects = append(ctx.session.PendingEffects, effectTask{
				instance:  ctx.instance,
				hookIndex: idx,
				fn:        fn,
			})
		}

		cell.deps = cloneDeps(deps)
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

// depsEqual compares dependency arrays with support for functions, channels, and maps.
// Functions and channels are compared by pointer identity (same function instance).
// Maps are compared by pointer (same map instance, not deep map equality).
// Other types use reflect.DeepEqual.
func depsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !depsValueEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func depsValueEqual(a, b any) bool {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	if !va.IsValid() && !vb.IsValid() {
		return true
	}

	if !va.IsValid() || !vb.IsValid() {
		return false
	}

	if va.Type() != vb.Type() {
		return false
	}

	switch va.Kind() {
	case reflect.Func, reflect.Chan:
		if va.IsNil() && vb.IsNil() {
			return true
		}
		if va.IsNil() || vb.IsNil() {
			return false
		}
		return va.Pointer() == vb.Pointer()

	case reflect.Map:
		if va.IsNil() && vb.IsNil() {
			return true
		}
		if va.IsNil() || vb.IsNil() {
			return false
		}
		return va.Pointer() == vb.Pointer()

	default:
		return reflect.DeepEqual(a, b)
	}
}

// UseErrorBoundary returns any error from child components.
// Returns nil if no child has errored.
// Components can use this to render custom error UI when children fail.
// When an error is caught, it is cleared from children to prevent propagation to parent boundaries.
func UseErrorBoundary(ctx *Ctx) *ComponentError {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeErrorBoundary,
			Value: (*ComponentError)(nil),
		})
	}

	slot := &ctx.instance.HookFrame[idx]
	if slot.Type != HookTypeErrorBoundary {
		panic("runtime: UseErrorBoundary hook mismatch")
	}

	childErr := ctx.instance.findChildError()

	if childErr != nil {
		ctx.instance.clearChildErrors()
	}

	slot.Value = childErr

	return childErr
}
