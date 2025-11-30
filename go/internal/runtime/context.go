package runtime

import (
	"fmt"
	"unsafe"
)

type contextID uintptr

type Context[T any] struct {
	id           contextID
	defaultValue T
	equal        func(a, b T) bool
}

func CreateContext[T any](defaultValue T) *Context[T] {
	ctx := &Context[T]{
		defaultValue: defaultValue,
	}
	ctx.id = contextID(uintptr(unsafe.Pointer(ctx)))
	return ctx
}

func (c *Context[T]) WithEqual(eq func(a, b T) bool) *Context[T] {
	c.equal = eq
	return c
}

func (c *Context[T]) UseProvider(ctx *Ctx, initial T) (T, func(T)) {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseProvider called outside component render")
	}

	inst := ctx.instance

	inst.mu.Lock()
	if inst.Providers == nil {
		inst.Providers = make(map[any]any)
	}

	existing, hasExisting := inst.Providers[c.id]
	if !hasExisting {
		inst.Providers[c.id] = initial
		existing = initial
	}
	inst.mu.Unlock()

	value, ok := existing.(T)
	if !ok {
		panic(fmt.Sprintf("runtime: context value type mismatch: expected %T, got %T (context ID: %v)", c.defaultValue, existing, c.id))
	}

	setter := c.createSetter(ctx, inst)

	return value, setter
}

func (c *Context[T]) UseContext(ctx *Ctx) (T, func(T)) {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseContext called outside component render")
	}

	providerInst, value := c.findProvider(ctx.instance)
	if providerInst == nil {
		return value, nil
	}

	setter := c.createSetter(ctx, providerInst)
	return value, setter
}

func (c *Context[T]) UseContextValue(ctx *Ctx) T {
	value, _ := c.UseContext(ctx)
	return value
}

func (c *Context[T]) findProvider(inst *Instance) (*Instance, T) {
	current := inst
	for current != nil {
		current.mu.Lock()
		providers := current.Providers
		current.mu.Unlock()

		if providers != nil {
			if val, ok := providers[c.id]; ok {

				typed, typeOk := val.(T)
				if !typeOk {
					panic(fmt.Sprintf("runtime: context value type mismatch: expected %T, got %T (context ID: %v)", c.defaultValue, val, c.id))
				}
				return current, typed
			}
		}
		current = current.Parent
	}

	return nil, c.defaultValue
}

func (c *Context[T]) createSetter(ctx *Ctx, providerInst *Instance) func(T) {
	eq := c.equal
	if eq == nil {
		eq = defaultEqual[T]()
	}

	return func(newValue T) {
		providerInst.mu.Lock()
		raw, exists := providerInst.Providers[c.id]
		if !exists {
			providerInst.mu.Unlock()
			return
		}

		oldValue, ok := raw.(T)
		if !ok {
			providerInst.mu.Unlock()

			providerInst.Providers[c.id] = newValue
			if ctx.session != nil {
				providerInst.NotifyContextChange(ctx.session)
			}
			return
		}

		equal := safeEqual(eq, oldValue, newValue)
		if equal {
			providerInst.mu.Unlock()
			return
		}

		providerInst.Providers[c.id] = newValue
		providerInst.mu.Unlock()

		if ctx.session != nil {
			providerInst.NotifyContextChange(ctx.session)
		}
	}
}

func safeEqual[T any](eq func(a, b T) bool, a, b T) (equal bool) {
	defer func() {
		if r := recover(); r != nil {
			equal = false
		}
	}()
	return eq(a, b)
}
