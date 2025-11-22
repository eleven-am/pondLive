package runtime

import (
	"fmt"
	"reflect"

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

// ComponentDepth returns the depth of the current component in the tree (root = 0).
func (c Ctx) ComponentDepth() int {
	if c.comp == nil {
		return 0
	}
	return c.comp.depth()
}

// IsLive returns true if the session is in websocket mode (transport connected).
// Returns false during SSR or if no session is available.
func (c Ctx) IsLive() bool {
	if c.sess == nil {
		return false
	}
	return c.sess.IsLive()
}

// Render renders a child component with optional key.
func Render[P any](ctx Ctx, fn Component[P], props P, opts ...RenderOption) *dom.StructuredNode {
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

	key := cfg.key
	if key == "" {

		key = fmt.Sprintf("auto:%d", ctx.comp.childRenderIndex)
		ctx.comp.childRenderIndex++
	}

	child := ctx.comp.ensureChild(adapter, key, props)
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
		child.wrapper = &dom.StructuredNode{
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

// NoPropsComponent wraps a component function that accepts children as a slice.
// The first top-level dom.Key() in children is extracted and used as the component's render key.
func NoPropsComponent[F any](fn func(Ctx, []dom.Item) *dom.StructuredNode, original F) func(Ctx, ...dom.Item) *dom.StructuredNode {
	if fn == nil {
		return nil
	}

	return func(ctx Ctx, children ...dom.Item) *dom.StructuredNode {

		key, remainingChildren := dom.ExtractKey(children)

		var opts []RenderOption
		if key != "" {
			opts = append(opts, WithKey(key))
		}

		wrapper := func(c Ctx, _ struct{}) *dom.StructuredNode {
			return fn(c, remainingChildren)
		}

		return Render(ctx, wrapper, struct{}{}, opts...)
	}
}

// PropsComponent wraps a component function that accepts props and children as a slice.
// The first top-level dom.Key() in children is extracted and used as the component's render key.
func PropsComponent[P any, F any](fn func(Ctx, P, []dom.Item) *dom.StructuredNode, original F) func(Ctx, P, ...dom.Item) *dom.StructuredNode {
	if fn == nil {
		return nil
	}

	return func(ctx Ctx, props P, children ...dom.Item) *dom.StructuredNode {

		key, remainingChildren := dom.ExtractKey(children)

		var opts []RenderOption
		if key != "" {
			opts = append(opts, WithKey(key))
		}

		wrapper := func(c Ctx, p P) *dom.StructuredNode {
			return fn(c, p, remainingChildren)
		}

		return Render(ctx, wrapper, props, opts...)
	}
}

// EnqueueDOMAction implements dom.Dispatcher by enqueuing a DOM action effect.
func (c Ctx) EnqueueDOMAction(effect dom.DOMActionEffect) {
	if c.sess == nil {
		return
	}
	c.sess.enqueueDOMAction(effect)
}

// DOMGet implements dom.Dispatcher by requesting property values from the client.
func (c Ctx) DOMGet(ref string, selectors ...string) (map[string]any, error) {
	if c.sess == nil {
		return nil, fmt.Errorf("runtime2: DOMGet requires session context")
	}
	return c.sess.domGet(ref, selectors...)
}

// DOMAsyncCall implements dom.Dispatcher by calling a method on the client and returning the result.
func (c Ctx) DOMAsyncCall(ref string, method string, args ...any) (any, error) {
	if c.sess == nil {
		return nil, fmt.Errorf("runtime2: DOMAsyncCall requires session context")
	}
	return c.sess.domAsyncCall(ref, method, args...)
}

// EnqueueNavigation queues a navigation update to be sent to the client.
func (c Ctx) EnqueueNavigation(href string, replace bool) {
	if c.sess == nil {
		return
	}
	c.sess.EnqueueNavigation(href, replace)
}

func panicHookMismatch(comp *component, idx int, expected string, actual any) {
	name := "<component>"
	if comp != nil && comp.callable != nil {
		name = comp.callable.name()
	}
	actualType := fmt.Sprintf("%T", actual)
	panic(fmt.Sprintf("runtime2: hooks mismatch in %s at index %d (expected %s, got %s)", name, idx, expected, actualType))
}
