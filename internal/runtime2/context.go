package runtime2

import (
	"fmt"
	"unsafe"
)

// contextID uniquely identifies a context type.
type contextID uintptr

// Context represents a typed context that can be provided and consumed in the component tree.
//
// Design notes:
//   - Single provider per component: Multiple UseProvider calls for the same context on one
//     component will overwrite. This is intentional - use different components for different providers.
//   - Provider doesn't self-dirty: When setter is called, only children are marked dirty.
//     If the provider component needs to re-render based on its own context value, use UseState
//     alongside UseProvider to track the value locally.
//   - SSR mode (session nil): The setter won't mark children dirty when session is nil.
//     This is expected for SSR where the tree is rendered once without live updates.
//   - Equality: Default uses reflect.DeepEqual which may cause extra re-renders for types
//     containing functions, maps, or channels. Use WithEqual for such types.
type Context[T any] struct {
	id           contextID
	defaultValue T
	equal        func(a, b T) bool
}

// CreateContext creates a new context with the given default value.
// The default value is returned by UseContext when no provider exists in the ancestor chain.
func CreateContext[T any](defaultValue T) *Context[T] {
	ctx := &Context[T]{
		defaultValue: defaultValue,
	}
	ctx.id = contextID(uintptr(unsafe.Pointer(ctx)))
	return ctx
}

// WithEqual sets a custom equality function for the context.
// This is useful for types that contain functions, maps, or channels which cannot be compared with reflect.DeepEqual.
// Without a custom equality function, types with uncomparable fields may cause unnecessary re-renders
// or panics (which are recovered and treated as "values different").
func (c *Context[T]) WithEqual(eq func(a, b T) bool) *Context[T] {
	c.equal = eq
	return c
}

// UseProvider provides a value to all descendants in the component tree.
// Returns the current value and a setter function.
// The setter performs deep equality check and only triggers re-renders if the value changed.
//
// Note: Calling UseProvider multiple times for the same context in one component will use
// the existing stored value (not overwrite with the new initial). The initial is only used
// on first render.
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

// UseContext retrieves the context value from the nearest ancestor provider.
// Returns the current value and a setter function.
// If no provider exists, returns the default value and a nil setter.
// Calling the setter updates the provider's value and triggers re-renders for all consumers.
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

// UseContextValue retrieves the context value from the nearest ancestor provider.
// Returns only the current value. If no provider exists, returns the default value.
// Use this when you only need to read the context value without updating it.
func (c *Context[T]) UseContextValue(ctx *Ctx) T {
	value, _ := c.UseContext(ctx)
	return value
}

// findProvider walks up the component tree to find the nearest provider.
// Returns the provider instance and the current value.
// If no provider is found, returns nil and the default value.
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

// createSetter creates a setter function that updates the provider value.
// The setter performs equality check and notifies children if the value changed.
//
// Note: The setter only marks children dirty, not the provider component itself.
// If session is nil (SSR mode), children won't be marked dirty.
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

// safeEqual calls the equality function with panic recovery.
// If the equality check panics, returns false (values treated as different).
// This handles uncomparable types gracefully when no custom equality is provided.
func safeEqual[T any](eq func(a, b T) bool, a, b T) (equal bool) {
	defer func() {
		if r := recover(); r != nil {
			equal = false
		}
	}()
	return eq(a, b)
}
