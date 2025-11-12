package runtime

import (
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/dom"
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

// RequestComponentBoot schedules a template refresh for the current component instance.
func (c Ctx) RequestComponentBoot() {
	if c.sess == nil || c.comp == nil {
		return
	}
	c.sess.RequestComponentBoot(c.comp.id)
}

// Render renders a child component with optional key.
func Render[P any](ctx Ctx, fn Component[P], props P, opts ...RenderOption) dom.Node {
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

// EnqueueDOMAction implements dom.Dispatcher by enqueuing a DOM action effect.
func (c Ctx) EnqueueDOMAction(effect dom.DOMActionEffect) {
	if c.sess == nil || c.sess.owner == nil {
		return
	}
	c.sess.owner.enqueueFrameEffect(effect)
}

// DOMGet implements dom.Dispatcher by requesting property values from the client.
func (c Ctx) DOMGet(ref string, selectors ...string) (map[string]any, error) {
	if c.sess == nil || c.sess.owner == nil {
		return nil, fmt.Errorf("runtime: domget requires live session context")
	}
	return c.sess.owner.DOMGet(ref, selectors...)
}

// DOMAsyncCall implements dom.Dispatcher by calling a method on the client and returning the result.
func (c Ctx) DOMAsyncCall(ref string, method string, args ...any) (any, error) {
	if c.sess == nil || c.sess.owner == nil {
		return nil, fmt.Errorf("runtime: domasynccall requires live session context")
	}
	return c.sess.owner.DOMAsyncCall(ref, method, args...)
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
