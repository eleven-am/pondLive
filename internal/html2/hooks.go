package html2

import (
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type (
	Ctx                       = runtime2.Ctx
	Ref[T any]                = runtime2.Ref[T]
	StateOpt[T any]           = runtime2.StateOpt[T]
	ComponentError            = runtime2.ComponentError
	ElementRef                = runtime2.ElementRef
	Context[T any]            = runtime2.Context[T]
	ScriptHandle              = runtime2.ScriptHandle
	HandlerFunc               = runtime2.HandlerFunc
	HandlerHandle             = runtime2.HandlerHandle
	SlotRenderer              = runtime2.SlotRenderer
	ScopedSlotRenderer[T any] = runtime2.ScopedSlotRenderer[T]
	UploadHandle              = runtime2.UploadHandle
	Styles                    = runtime2.Styles
	StreamItem[T any]         = runtime2.StreamItem[T]
	StreamHandle[T any]       = runtime2.StreamHandle[T]
)

func CreateContext[T any](defaultValue T) *Context[T] {
	return runtime2.CreateContext(defaultValue)
}

func WithEqual[T any](eq func(a, b T) bool) StateOpt[T] {
	return runtime2.WithEqual(eq)
}

func UseState[T any](ctx *Ctx, initial T, opts ...StateOpt[T]) (T, func(T)) {
	return runtime2.UseState(ctx, initial, opts...)
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
	return runtime2.UseRef(ctx, initial)
}

func UseElement(ctx *Ctx) *ElementRef {
	return runtime2.UseElement(ctx)
}

func UseMemo[T any](ctx *Ctx, compute func() T, deps ...any) T {
	return runtime2.UseMemo(ctx, compute, deps...)
}

func UseEffect(ctx *Ctx, fn func() func(), deps ...any) {
	runtime2.UseEffect(ctx, fn, deps...)
}

func UseErrorBoundary(ctx *Ctx) *ComponentError {
	return runtime2.UseErrorBoundary(ctx)
}

func UseScript(ctx *Ctx, script string) ScriptHandle {
	return runtime2.UseScript(ctx, script)
}

func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	return runtime2.UseHandler(ctx, method, chain...)
}

func UseSlots(ctx *Ctx, children []work.Node) *SlotRenderer {
	return runtime2.UseSlots(ctx, children)
}

func UseScopedSlots[T any](ctx *Ctx, children []work.Node) *ScopedSlotRenderer[T] {
	return runtime2.UseScopedSlots[T](ctx, children)
}

func UseUpload(ctx *Ctx) UploadHandle {
	return runtime2.UseUpload(ctx)
}

func UseStream[T any](ctx *Ctx, renderRow func(StreamItem[T]) Node, initial ...StreamItem[T]) (Node, StreamHandle[T]) {
	return runtime2.UseStream(ctx, renderRow, initial...)
}

func UseStyles(ctx *Ctx, rawCSS string) *Styles {
	return runtime2.UseStyles(ctx, rawCSS)
}
