package live

import (
	"context"

	"github.com/eleven-am/go/pondlive/internal/runtime"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

type (
	Ctx                  = runtime.Ctx
	Component[P any]     = runtime.Component[P]
	RenderOption         = runtime.RenderOption
	StateOpt[T any]      = runtime.StateOpt[T]
	Cleanup              = runtime.Cleanup
	Ref[T any]           = runtime.Ref[T]
	Node                 = h.Node
	Context[T any]       = runtime.Context[T]
	SessionID            = runtime.SessionID
	Session              = runtime.ComponentSession
	Meta                 = runtime.Meta
	RenderResult         = runtime.RenderResult
	PubsubMessage[T any] = runtime.PubsubMessage[T]
	PubsubHandle[T any]  = runtime.PubsubHandle[T]
	PubsubPublisher      = runtime.PubsubPublisher
	PubsubPublishFunc    = runtime.PubsubPublishFunc
	PubsubOption[T any]  = runtime.PubsubOption[T]
	Pubsub[T any]        = runtime.Pubsub[T]
)

// Render invokes the supplied child component with props, returning its node.
// Use it within your component to manually compose children. Combine with
// WithKey to give siblings stable identities in lists.
func Render[P any](ctx Ctx, fn Component[P], props P, opts ...RenderOption) h.Node {
	return runtime.Render(ctx, fn, props, opts...)
}

// WithKey assigns a deterministic key to a child rendered via Render. Helpful
// when rendering slices so LiveUI can diff elements predictably.
func WithKey(key string) RenderOption { return runtime.WithKey(key) }

// UseState creates reactive state scoped to the component. It returns getter
// and setter closures; calling the setter schedules a rerender. Supply
// WithEqual to suppress renders when the value hasn’t meaningfully changed.
func UseState[T any](ctx Ctx, initial T, opts ...StateOpt[T]) (func() T, func(T)) {
	return runtime.UseState(ctx, initial, opts...)
}

// UseMemo memoizes compute until any dependency changes. It’s useful for
// expensive calculations or deriving values from props/state without
// recomputing every render.
func UseMemo[T any](ctx Ctx, compute func() T, deps ...any) T {
	return runtime.UseMemo(ctx, compute, deps...)
}

// UseEffect runs setup after render and optionally returns a cleanup that runs
// on dependency change or unmount. Provide deps to limit when the effect
// re-executes.
func UseEffect(ctx Ctx, setup func() Cleanup, deps ...any) {
	runtime.UseEffect(ctx, setup, deps...)
}

// UseRef returns a pointer holding mutable state that persists across renders.
// It’s ideal for tracking DOM handles or other imperative data.
func UseRef[T any](ctx Ctx, zero T) *Ref[T] {
	return runtime.UseRef(ctx, zero)
}

// WithEqual customizes UseState comparisons. If eq(old, new) is true, the
// setter skips scheduling a rerender.
func WithEqual[T any](eq func(a, b T) bool) StateOpt[T] {
	return runtime.WithEqual(eq)
}

// UseSelect subscribes to a context value, projecting it with pick. The eq
// function controls whether the projected value changed, avoiding unnecessary
// rerenders when unrelated context fields update.
func UseSelect[T any, U any](ctx Ctx, c Context[T], pick func(T) U, eq func(U, U) bool) U {
	return runtime.UseSelect(ctx, c, pick, eq)
}

// NewContext creates a context handle with a default value. Use Provide on the
// returned context to supply overrides, and Use to read it down the tree.
func NewContext[T any](def T) Context[T] {
	return runtime.NewContext(def)
}

// WithMetadata couples a node with document metadata (title, meta tags, etc.).
// Use it in layouts or top-level pages that set head information.
func WithMetadata(body h.Node, meta *Meta) *RenderResult {
	return runtime.WithMetadata(body, meta)
}

// MergeMeta merges metadata structs, preferring non-empty fields in overrides
// and appending tag slices. Handy when combining layout- and page-level meta.
func MergeMeta(base *Meta, overrides ...*Meta) *Meta {
	return runtime.MergeMeta(base, overrides...)
}

func NewPubsub[T any](topic string, publisher PubsubPublisher, opts ...PubsubOption[T]) *Pubsub[T] {
	return runtime.NewPubsub(topic, publisher, opts...)
}

func WithPubsubCodec[T any](encode func(T) ([]byte, error), decode func([]byte) (T, error)) PubsubOption[T] {
	return runtime.WithPubsubCodec(encode, decode)
}

func WithPubsubProvider[T any](provider runtime.PubsubProvider) PubsubOption[T] {
	return runtime.WithPubsubProvider[T](provider)
}

func WrapPubsubProvider(provider runtime.PubsubProvider) func(context.Context, string, []byte, map[string]string) error {
	return runtime.WrapPubsubProvider(provider)
}
