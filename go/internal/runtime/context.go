package runtime

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// contextID uniquely identifies a context type.
type contextID uintptr

// Context represents a typed context that can provide values to a component tree.
type Context[T any] struct {
	id           contextID
	defaultValue T
}

// CreateContext creates a new context with a default value.
// Each context instance has a unique ID derived from its pointer address.
func CreateContext[T any](defaultValue T) *Context[T] {
	ctx := &Context[T]{
		defaultValue: defaultValue,
	}

	ctx.id = contextID(uintptr(unsafe.Pointer(ctx)))
	return ctx
}

// Provide renders children with this context value available.
// The value is provided to all descendants until another provider overrides it.
// Creates a component boundary to scope the provider value.
func (c *Context[T]) Provide(ctx Ctx, value T, children func(Ctx) *dom2.StructuredNode) *dom2.StructuredNode {
	if ctx.comp == nil {
		panic("runtime2: Context.Provide called outside component render")
	}

	type providerProps struct {
		contextID contextID
		value     any
		children  func(Ctx) *dom2.StructuredNode
	}

	provider := func(pctx Ctx, props providerProps) *dom2.StructuredNode {
		if pctx.comp.providers == nil {
			pctx.comp.providers = make(map[contextID]any)
		}
		prev, ok := pctx.comp.providers[props.contextID]
		if !ok || !reflect.DeepEqual(prev, props.value) {
			pctx.comp.notifyContextChange()
		}
		pctx.comp.providers[props.contextID] = props.value
		return props.children(pctx)
	}

	seq := ctx.comp.providerSeq
	ctx.comp.providerSeq++
	key := fmt.Sprintf("ctx:%d:%d", c.id, seq)

	return Render(ctx, provider, providerProps{
		contextID: c.id,
		value:     value,
		children:  children,
	}, WithKey(key))
}

// Use reads the nearest provided value for this context.
// Walks up the component tree until a provider is found.
// Returns the default value if no provider exists.
func (c *Context[T]) Use(ctx Ctx) T {
	if ctx.comp == nil {
		panic("runtime2: Context.Use called outside component render")
	}

	current := ctx.comp
	for current != nil {
		if current.providers != nil {
			if val, ok := current.providers[c.id]; ok {
				return val.(T)
			}
		}
		current = current.parent
	}

	return c.defaultValue
}
