package runtime

import (
	"fmt"
	"reflect"
	"runtime/debug"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ref[T any] struct {
	Current T
}

type StateOpt[T any] interface{ applyStateOpt(*stateCell[T]) }

type stateOptFunc[T any] func(*stateCell[T])

func (f stateOptFunc[T]) applyStateOpt(c *stateCell[T]) { f(c) }

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

func UseMemo[T any](ctx *Ctx, compute func() T, deps ...any) T {
	idx := ctx.hookIndex
	ctx.hookIndex++

	safeCompute := func() (result T, err *Error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())

				var parentID string
				if ctx.instance.Parent != nil {
					parentID = ctx.instance.Parent.ID
				}

				var sessionID string
				var devMode bool
				if ctx.session != nil {
					sessionID = ctx.session.SessionID
					devMode = ctx.session.devMode
				}

				ectx := ErrorContext{
					SessionID:         sessionID,
					ComponentID:       ctx.instance.ID,
					ComponentName:     ctx.instance.ComponentName(),
					ParentID:          parentID,
					ComponentPath:     ctx.instance.BuildComponentPath(),
					ComponentNamePath: ctx.instance.BuildComponentNamePath(),
					Phase:             "memo",
					HookIndex:         idx,
					HookCount:         len(ctx.instance.HookFrame),
					Props:             ctx.instance.Props,
					ProviderKeys:      ctx.instance.GetProviderKeys(),
					DevMode:           devMode,
				}
				err = NewComponentErrorWithContext(ErrCodeMemo, fmt.Sprintf("%v", r), stack, ectx)
				err.Meta["panic_value"] = r

				if ctx.session != nil && ctx.session.devMode && ctx.session.Bus != nil {
					ctx.session.Bus.ReportDiagnostic(protocol.Diagnostic{
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
			ctx.instance.setRenderError(err)
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
			ctx.instance.setRenderError(err)
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
	hasDeps bool
}

func UseEffect(ctx *Ctx, fn func() func(), deps ...any) {
	idx := ctx.hookIndex
	ctx.hookIndex++

	hasDeps := deps != nil

	if idx >= len(ctx.instance.HookFrame) {
		cell := &effectCell{cleanup: nil, deps: cloneDeps(deps), hasDeps: hasDeps}
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

	if !hasDeps {
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

func UseErrorBoundary(ctx *Ctx) *ErrorBatch {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeErrorBoundary,
			Value: (*ErrorBatch)(nil),
		})
	}

	slot := &ctx.instance.HookFrame[idx]
	if slot.Type != HookTypeErrorBoundary {
		panic("runtime: UseErrorBoundary hook mismatch")
	}

	renderErrors := ctx.instance.collectChildErrors()
	effectErrors := ctx.instance.collectChildEffectErrors()

	allErrors := append(renderErrors, effectErrors...)

	if len(allErrors) > 0 {
		ctx.instance.clearChildErrors()
		ctx.instance.clearChildEffectErrors()
		batch := NewErrorBatch(allErrors...)
		slot.Value = batch
		return batch
	}

	slot.Value = nil
	return nil
}

type channelCell struct {
	channel *Channel
}

func UseChannel(ctx *Ctx, channelName string) *Channel {
	if ctx == nil || ctx.instance == nil || ctx.session == nil {
		return nil
	}

	idx := ctx.hookIndex
	ctx.hookIndex++

	isMount := idx >= len(ctx.instance.HookFrame)

	if isMount {
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeChannel,
			Value: &channelCell{},
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*channelCell)
	if !ok {
		panic("runtime: UseChannel hook mismatch")
	}

	if isMount {
		mgr := ctx.session.ChannelManager()
		if mgr == nil {
			return nil
		}

		ref := mgr.Join(channelName)
		ch := newChannel(ref)
		cell.channel = ch

		sub := ch.subscribeToBus(ctx.session.Bus)

		ctx.instance.RegisterCleanup(func() {
			if sub != nil {
				sub.Unsubscribe()
			}
			mgr.Leave(channelName)
		})
	}

	cell.channel.resetHandlers()

	return cell.channel
}
