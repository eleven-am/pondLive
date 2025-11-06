package runtime

import (
	"fmt"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// Ctx represents the runtime context passed to every component render call.
type Ctx struct {
	sess  *ComponentSession
	comp  *component
	frame *hookFrame
}

// Session returns the backing runtime session.
func (c Ctx) Session() *ComponentSession { return c.sess }

// ComponentID returns the unique runtime id for the current component instance.
func (c Ctx) ComponentID() string {
	if c.comp == nil {
		return ""
	}
	return c.comp.id
}

// Render renders a child component with optional key.
func Render[P any](ctx Ctx, fn Component[P], props P, opts ...RenderOption) h.Node {
	if ctx.sess == nil || ctx.comp == nil {
		panic("runtime: render called outside component context")
	}
	var cfg renderOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyRenderOption(&cfg)
		}
	}
	adapter := newComponentAdapter(fn)
	child := ctx.comp.ensureChild(adapter, cfg.key, props)
	child.props = props
	child.callable = adapter
	child.sess = ctx.sess
	return child.render()
}

// renderOptions configure child component rendering.
type renderOptions struct {
	key string
}

// RenderOption configures Render behaviour.
type RenderOption interface{ applyRenderOption(*renderOptions) }

type renderOptionFunc func(*renderOptions)

func (f renderOptionFunc) applyRenderOption(o *renderOptions) { f(o) }

// WithKey assigns a stable key to the rendered child component.
func WithKey(key string) RenderOption {
	return renderOptionFunc(func(o *renderOptions) { o.key = key })
}

// WithoutRender batches state updates without triggering an immediate flush.
func WithoutRender(ctx Ctx, fn func()) {
	if ctx.sess == nil {
		fn()
		return
	}
	ctx.sess.suspend++
	defer func() {
		ctx.sess.suspend--
		if ctx.sess.suspend < 0 {
			ctx.sess.suspend = 0
		}
	}()
	fn()
}

// NoRender clears the dirty flag for the current component this tick.
func NoRender(ctx Ctx) {
	if ctx.sess == nil || ctx.comp == nil {
		return
	}
	if ctx.sess.currentComponent() == ctx.comp {
		ctx.comp.dirty = false
	} else {
		ctx.comp.markClean()
	}
	ctx.sess.clearDirty(ctx.comp)
}

func panicHookMismatch(comp *component, idx int, expected string, actual any) {
	name := "<component>"
	if comp != nil && comp.callable != nil {
		name = comp.callable.name()
	}
	actualType := fmt.Sprintf("%T", actual)
	panic(hookMismatchError{
		message:   fmt.Sprintf("runtime: hooks mismatch in %s at index %d (expected %s, got %s)", name, idx, expected, actualType),
		hook:      expected,
		index:     idx,
		actual:    actualType,
		component: name,
	})
}

type hookMismatchError struct {
	message   string
	hook      string
	index     int
	actual    string
	component string
}

func (e hookMismatchError) Error() string { return e.message }

func (e hookMismatchError) Metadata() map[string]any {
	return map[string]any{
		"expectedHook": e.hook,
		"actualType":   e.actual,
		"hookIndex":    e.index,
		"component":    e.component,
	}
}

func (e hookMismatchError) Suggestion() string {
	return "Ensure hooks are invoked in the same order on every render and that hook calls are not conditional."
}

func (e hookMismatchError) HookName() string { return e.hook }

func (e hookMismatchError) HookIndex() int { return e.index }
