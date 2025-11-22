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
		ptr := cfg.origPointer
		if ptr == 0 {
			ptr = adapter.pointer()
		}
		key = fmt.Sprintf("auto:%x", ptr)
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
	key         string
	origPointer uintptr // Original function pointer for auto-key generation
}

type RenderOption interface{ applyRenderOption(*renderOptions) }

type renderOptionFunc func(*renderOptions)

func (f renderOptionFunc) applyRenderOption(o *renderOptions) { f(o) }

// WithKey assigns a stable key to the rendered child component.
func WithKey(key string) RenderOption {
	return renderOptionFunc(func(o *renderOptions) { o.key = key })
}

// withOriginalPointer is internal - stores the original function pointer for auto-key
func withOriginalPointer(ptr uintptr) RenderOption {
	return renderOptionFunc(func(o *renderOptions) { o.origPointer = ptr })
}

// NoPropsComponent is like NoPropsComponent but takes an additional
// original function parameter to use for stable key generation
func NoPropsComponent[F any](fn func(Ctx) *dom.StructuredNode, original F) func(Ctx, ...RenderOption) *dom.StructuredNode {
	if fn == nil {
		return nil
	}

	origPtr := reflect.ValueOf(original).Pointer()

	wrapped := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return fn(ctx)
	}
	return func(ctx Ctx, opts ...RenderOption) *dom.StructuredNode {

		opts = append(opts, withOriginalPointer(origPtr))
		return Render(ctx, wrapped, struct{}{}, opts...)
	}
}

// PropsComponent is like PropsComponent but takes an additional
// original function parameter to use for stable key generation
func PropsComponent[P any, F any](fn func(Ctx, P) *dom.StructuredNode, original F) func(Ctx, P, ...RenderOption) *dom.StructuredNode {
	if fn == nil {
		return nil
	}

	origPtr := reflect.ValueOf(original).Pointer()

	return func(ctx Ctx, props P, opts ...RenderOption) *dom.StructuredNode {

		opts = append(opts, withOriginalPointer(origPtr))
		return Render(ctx, fn, props, opts...)
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
