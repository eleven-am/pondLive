package pkg

import (
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/metatags"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type (
	Ctx                       = runtime.Ctx
	Ref[T any]                = runtime.Ref[T]
	StateOpt[T any]           = runtime.StateOpt[T]
	ComponentError            = runtime.ComponentError
	ElementRef                = runtime.ElementRef
	Context[T any]            = runtime.Context[T]
	ScriptHandle              = runtime.ScriptHandle
	HandlerFunc               = runtime.HandlerFunc
	HandlerHandle             = runtime.HandlerHandle
	SlotRenderer              = runtime.SlotRenderer
	ScopedSlotRenderer[T any] = runtime.ScopedSlotRenderer[T]
	UploadHandle              = runtime.UploadHandle
	Styles                    = runtime.Styles
	StreamItem[T any]         = runtime.StreamItem[T]
	StreamHandle[T any]       = runtime.StreamHandle[T]
	Meta                      = metatags.Meta
)

func CreateContext[T any](defaultValue T) *Context[T] {
	return runtime.CreateContext(defaultValue)
}

func WithEqual[T any](eq func(a, b T) bool) StateOpt[T] {
	return runtime.WithEqual(eq)
}

func UseState[T any](ctx *Ctx, initial T, opts ...StateOpt[T]) (T, func(T)) {
	return runtime.UseState(ctx, initial, opts...)
}

func UseProvider[T any](c *Context[T], ctx *Ctx, initial T) (T, func(T)) {
	return c.UseProvider(ctx, initial)
}

func UseContext[T any](c *Context[T], ctx *Ctx) (T, func(T)) {
	return c.UseContext(ctx)
}

func UseContextValue[T any](c *Context[T], ctx *Ctx) T {
	return c.UseContextValue(ctx)
}

func UseRef[T any](ctx *Ctx, initial T) *Ref[T] {
	return runtime.UseRef(ctx, initial)
}

func UseElement(ctx *Ctx) *ElementRef {
	return runtime.UseElement(ctx)
}

func UseMemo[T any](ctx *Ctx, compute func() T, deps ...any) T {
	return runtime.UseMemo(ctx, compute, deps...)
}

func UseEffect(ctx *Ctx, fn func() func(), deps ...any) {
	runtime.UseEffect(ctx, fn, deps...)
}

func UseErrorBoundary(ctx *Ctx) *ComponentError {
	return runtime.UseErrorBoundary(ctx)
}

func UseScript(ctx *Ctx, script string) ScriptHandle {
	return runtime.UseScript(ctx, script)
}

func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	return runtime.UseHandler(ctx, method, chain...)
}

func UseSlots(ctx *Ctx, children []work.Node) *SlotRenderer {
	return runtime.UseSlots(ctx, children)
}

func UseScopedSlots[T any](ctx *Ctx, children []work.Node) *ScopedSlotRenderer[T] {
	return runtime.UseScopedSlots[T](ctx, children)
}

func UseUpload(ctx *Ctx) UploadHandle {
	return runtime.UseUpload(ctx)
}

func UseStream[T any](ctx *Ctx, renderRow func(StreamItem[T]) Node, initial ...StreamItem[T]) (Node, StreamHandle[T]) {
	return runtime.UseStream(ctx, renderRow, initial...)
}

func UseStyles(ctx *Ctx, rawCSS string) *Styles {
	return runtime.UseStyles(ctx, rawCSS)
}

func UseHeaders(ctx *Ctx) http.Header {
	return headers.UseHeaders(ctx)
}

func UseMetaTags(ctx *Ctx, meta *Meta) {
	metatags.UseMetaTags(ctx, meta)
}
