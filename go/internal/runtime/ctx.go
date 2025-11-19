package runtime

import (
	"fmt"
	"reflect"

	"github.com/eleven-am/pondlive/go/internal/dom2"
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
func Render[P any](ctx Ctx, fn Component[P], props P, opts ...RenderOption) *dom2.StructuredNode {
	if ctx.sess == nil || ctx.comp == nil {
		panic("runtime2: render called outside component context")
	}
	var cfg renderOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyRenderOption(&cfg)
		}
	}
	adapter := newComponentAdapter(fn)
	child := ctx.comp.ensureChild(adapter, cfg.key, props)
	child.callable = adapter
	child.sess = ctx.sess

	shouldRender := true
	parentEpoch := ctx.comp.combinedContextEpoch
	if child.parentContextEpoch != parentEpoch {
		shouldRender = true
	} else if !child.dirty && child.prevProps != nil {
		if reflect.DeepEqual(child.prevProps, props) {
			shouldRender = false
		}
	}

	if shouldRender {
		child.prevProps = props
	}
	child.props = props

	if child.wrapper == nil {
		child.wrapper = &dom2.StructuredNode{
			ComponentID: child.id,
		}
	}

	if shouldRender {
		child.render()
	}

	child.parentContextEpoch = parentEpoch
	return child.wrapper
}

type renderOptions struct {
	key string
}

type RenderOption interface{ applyRenderOption(*renderOptions) }

type renderOptionFunc func(*renderOptions)

func (f renderOptionFunc) applyRenderOption(o *renderOptions) { f(o) }

// WithKey assigns a stable key to the rendered child component.
func WithKey(key string) RenderOption {
	return renderOptionFunc(func(o *renderOptions) { o.key = key })
}

func NoPropsComponent(fn func(Ctx) *dom2.StructuredNode) func(Ctx, ...RenderOption) *dom2.StructuredNode {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return fn(ctx)
	}
	return func(ctx Ctx, opts ...RenderOption) *dom2.StructuredNode {
		return Render(ctx, wrapped, struct{}{}, opts...)
	}
}

func PropsComponent[P any](fn func(Ctx, P) *dom2.StructuredNode) func(Ctx, P, ...RenderOption) *dom2.StructuredNode {
	if fn == nil {
		return nil
	}
	return func(ctx Ctx, props P, opts ...RenderOption) *dom2.StructuredNode {
		return Render(ctx, fn, props, opts...)
	}
}

// EnqueueDOMAction implements dom2.Dispatcher by enqueuing a DOM action effect.
func (c Ctx) EnqueueDOMAction(effect dom2.DOMActionEffect) {
	if c.sess == nil {
		return
	}
	c.sess.enqueueDOMAction(effect)
}

// DOMGet implements dom2.Dispatcher by requesting property values from the client.
func (c Ctx) DOMGet(ref string, selectors ...string) (map[string]any, error) {
	if c.sess == nil {
		return nil, fmt.Errorf("runtime2: DOMGet requires session context")
	}
	return c.sess.domGet(ref, selectors...)
}

// DOMAsyncCall implements dom2.Dispatcher by calling a method on the client and returning the result.
func (c Ctx) DOMAsyncCall(ref string, method string, args ...any) (any, error) {
	if c.sess == nil {
		return nil, fmt.Errorf("runtime2: DOMAsyncCall requires session context")
	}
	return c.sess.domAsyncCall(ref, method, args...)
}

func panicHookMismatch(comp *component, idx int, expected string, actual any) {
	name := "<component>"
	if comp != nil && comp.callable != nil {
		name = comp.callable.name()
	}
	actualType := fmt.Sprintf("%T", actual)
	panic(fmt.Sprintf("runtime2: hooks mismatch in %s at index %d (expected %s, got %s)", name, idx, expected, actualType))
}
