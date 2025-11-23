package live

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/dom/css"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
	"github.com/eleven-am/pondlive/go/internal/slot"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type (
	Ctx                               = runtime.Ctx
	RenderOption                      = runtime.RenderOption
	StateOpt[T any]                   = runtime.StateOpt[T]
	Cleanup                           = runtime.Cleanup
	Ref[T any]                        = runtime.Ref[T]
	ElementRef[T h.ElementDescriptor] = h.ElementRef[T]
	ElementDescriptor                 = h.ElementDescriptor
	Node                              = h.Node
	Context[T any]                    = runtime.Context[T]
	SessionID                         = session.SessionID
	Session                           = runtime.ComponentSession
	ScrollOptions                     = dom.ScrollOptions
	PubsubMessage[T any]              = runtime.PubsubMessage[T]
	PubsubHandle[T any]               = runtime.PubsubHandle[T]
	PubsubOption[T any]               = runtime.PubsubOption[T]
	StreamItem[T any]                 = runtime.StreamItem[T]
	StreamHandle[T any]               = runtime.StreamHandle[T]
	RuntimeComponent[P any]           = runtime.Component[P]
	NavMsg                            = router.NavMsg
	PopMsg                            = router.PopMsg
	Styles                            = runtime.Styles
	ScriptHandle                      = runtime.ScriptHandle
	UploadHandle                      = runtime.UploadHandle
	UploadConfig                      = runtime.UploadConfig
	UploadEvent                       = runtime.UploadEvent
	HeadersHandle                     = headers.Handle
	CookieOptions                     = headers.CookieOptions
	SlotContext                       = slot.SlotContext
	ScopedSlotContext[T any]          = slot.ScopedSlotContext[T]
)

// Component wraps a component function that accepts children as a slice.
// Children can include h.Key() at the top level to set the component's render key.
//
// Example:
//
//	card := Component(func(ctx Ctx, children []h.Item) h.Node {
//	    return h.Div(h.H1(h.Text("Card")), h.Fragment(children...))
//	})
//	card(ctx, h.Key("my-card"), h.Text("Child 1"))
func Component(fn func(Ctx, []h.Item) h.Node) func(Ctx, ...h.Item) h.Node {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx, children []dom.Item) *dom.StructuredNode {
		return fn(ctx, children)
	}
	return runtime.NoPropsComponent(wrapped, fn)
}

// PropsComponent wraps a component function that accepts props and children.
//
// Example:
//
//	card := PropsComponent(func(ctx Ctx, props CardProps, children []h.Item) h.Node {
//	    return h.Div(h.H1(h.Text(props.Title)), h.Fragment(children...))
//	})
//	card(ctx, CardProps{Title: "Inbox"}, h.Key("my-card"), h.Text("Message"))
func PropsComponent[P any](fn func(Ctx, P, []h.Item) h.Node) func(Ctx, P, ...h.Item) h.Node {
	if fn == nil {
		return nil
	}
	wrapped := func(ctx Ctx, props P, children []dom.Item) *dom.StructuredNode {
		return fn(ctx, props, children)
	}
	return runtime.PropsComponent(wrapped, fn)
}

// Render invokes a child component with props. Deprecated: use Component or PropsComponent.
func Render[P any](ctx Ctx, fn RuntimeComponent[P], props P, opts ...RenderOption) h.Node {
	return runtime.Render(ctx, fn, props, opts...)
}

// WithKey assigns a key to a child rendered via Render.
func WithKey(key string) RenderOption { return runtime.WithKey(key) }

// UseState creates reactive state. Returns getter and setter; setter schedules a render.
// Example: count, setCount := UseState(ctx, 0)
func UseState[T any](ctx Ctx, initial T, opts ...StateOpt[T]) (func() T, func(T)) {
	return runtime.UseState(ctx, initial, opts...)
}

// UseMemo memoizes a computation until dependencies change.
// Example: filtered := UseMemo(ctx, func() []Product { return filter(products()) }, products())
func UseMemo[T any](ctx Ctx, compute func() T, deps ...any) T {
	return runtime.UseMemo(ctx, compute, deps...)
}

// UseEffect runs setup after render, returns optional cleanup.
// Example: UseEffect(ctx, func() Cleanup { ticker := time.NewTicker(1*time.Second); return ticker.Stop })
func UseEffect(ctx Ctx, setup func() Cleanup, deps ...any) {
	runtime.UseEffect(ctx, setup, deps...)
}

// UseRef returns mutable state that persists across renders without triggering rerenders.
// Example: prevValue := UseRef(ctx, 0); prevValue.Cur = newValue
func UseRef[T any](ctx Ctx, zero T) *Ref[T] {
	return runtime.UseRef(ctx, zero)
}

type hookable[R any] interface {
	HookBuild(any) R
}

// UseElement returns an HTML element ref for attaching events and calling DOM methods.
// Example: btnRef := UseElement[*h.ButtonRef](ctx); return h.Button(h.Attach(btnRef))
func UseElement[R hookable[R]](ctx Ctx) R {
	var zero R
	return zero.HookBuild(ctx)
}

// UseStream renders and manages a keyed list with mutation helpers.
// Example: node, handle := UseStream(ctx, func(item StreamItem[Todo]) h.Node { return h.Li(...) })
func UseStream[T any](ctx Ctx, renderRow func(StreamItem[T]) h.Node, initial ...StreamItem[T]) (h.Node, StreamHandle[T]) {
	return runtime.UseStream(ctx, renderRow, initial...)
}

// WithEqual customizes UseState equality checks to avoid unnecessary rerenders.
func WithEqual[T any](eq func(a, b T) bool) StateOpt[T] {
	return runtime.WithEqual(eq)
}

// NewContext creates a context with a default value.
func NewContext[T any](def T) *Context[T] {
	return runtime.CreateContext(def)
}

// UseStyles parses CSS and returns scoped class names.
// Example: styles := UseStyles(ctx, `.card { padding: 16px; }`); return h.Div(h.Class(styles.Class("card")))
func UseStyles(ctx Ctx, css string) *Styles {
	return runtime.UseStyles(ctx, css)
}

// UseScript creates a client-side script with bidirectional server communication.
// Example: script := UseScript(ctx, `(el, t) => t.send({data})`); script.OnMessage(func(m) {...})
func UseScript(ctx Ctx, script string) ScriptHandle {
	return runtime.UseScript(ctx, script)
}

// UseUpload manages file uploads with progress tracking and server-side processing.
// Example: upload := UseUpload(ctx); upload.Accept(UploadConfig{MaxSize: 10*1024*1024})
func UseUpload(ctx Ctx) UploadHandle {
	return runtime.UseUpload(ctx)
}

// UseHeaders provides access to request headers and cookie management.
// Example: headers := UseHeaders(ctx); token, ok := headers.GetCookie("auth_token")
func UseHeaders(ctx Ctx) HeadersHandle {
	return headers.UseHeaders(ctx)
}

// CN (class name) intelligently merges Tailwind CSS classes, resolving conflicts
// by keeping the last occurrence. Similar to shadcn/ui's cn function.
//
// Example:
//
//	CN("px-4", "px-2") // → "px-2"
//	CN("rounded-md px-3 py-2", "bg-blue-500", className) // allows overrides
func CN(classes ...string) string {
	return css.CN(classes...)
}

// CreateSlotContext creates a new slot context for component slot management.
// Example: var cardSlots = CreateSlotContext(); cardSlots.Render(ctx, "header")
func CreateSlotContext() *SlotContext {
	return slot.CreateSlotContext()
}

// CreateScopedSlotContext creates a slot context that passes data to slots.
// Example: var tableSlots = CreateScopedSlotContext[Row](); tableSlots.Render(ctx, "actions", row)
func CreateScopedSlotContext[T any]() *ScopedSlotContext[T] {
	return slot.CreateScopedSlotContext[T]()
}

// Slot creates a named slot marker for providing content to specific slots.
// Example: Card(ctx, props, Slot("header", h.H2(h.Text("Title"))), h.P(h.Text("Body")))
func Slot(name string, children ...h.Item) h.Item {
	return slot.Slot(name, children...)
}

// ScopedSlot creates a scoped slot that receives data from the component.
// Example: Table(ctx, props, ScopedSlot("actions", func(row Row) h.Node { return h.Button(...) }))
func ScopedSlot[T any](name string, fn func(T) h.Node) h.Item {
	return slot.ScopedSlot[T](name, fn)
}
