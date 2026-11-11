package pkg

import (
	"github.com/eleven-am/pondlive/internal/css"
	"github.com/eleven-am/pondlive/internal/head"
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/shared"
)

func WithEqual[T any](eq func(a, b T) bool) SignalOpt[T] {
	return runtime.WithEqual(eq)
}

func UseSignal[T any](ctx *Ctx, initial T, opts ...SignalOpt[T]) *Signal[T] {
	return runtime.UseSignal(ctx, initial, opts...)
}

func UseComputed[T any](ctx *Ctx, fn func() T, opts ...SignalOpt[T]) ReadSignal[T] {
	return runtime.UseComputed(ctx, fn, opts...)
}

func Derive[A, B any](src ReadSignal[A], fn func(A) B) ReadSignal[B] {
	return runtime.Derive(src, fn)
}

func UseEffect(ctx *Ctx, fn func() func()) {
	runtime.UseEffect(ctx, fn)
}

func UseRenderEffect(ctx *Ctx, fn func() func()) {
	runtime.UseRenderEffect(ctx, fn)
}

func Const[T any](v T) ReadSignal[T] {
	return runtime.Const(v)
}

func NewContext[T any]() *Context[T] {
	return runtime.NewContext[T]()
}

func Provide[T any](ctx *Ctx, c *Context[T], s *Signal[T]) {
	runtime.Provide(ctx, c, s)
}

func UseContext[T any](ctx *Ctx, c *Context[T]) *Signal[T] {
	return runtime.UseContext(ctx, c)
}

func NewShared[T any](v T, opts ...SharedOpt) *Shared[T] {
	return shared.NewShared(v, opts...)
}

func NewSharedFamily[K comparable, T any](init func(K) T, opts ...SharedOpt) *SharedFamily[K, T] {
	return shared.NewSharedFamily(init, opts...)
}

func UseShared[T any](ctx *Ctx, s *Shared[T]) ReadSignal[T] {
	return runtime.UseShared(ctx, s)
}

func NewBroadcast[T any](opts ...BroadcastOpt) *Broadcast[T] {
	return shared.NewBroadcast[T](opts...)
}

func UseBroadcast[T any](ctx *Ctx, b *Broadcast[T], handler func(T)) {
	runtime.UseBroadcast(ctx, b, handler)
}

func UseScript(ctx *Ctx, src string) Script {
	return runtime.UseScript(ctx, src)
}

func ScopeCSS(key, rawCSS string) *ScopedStyles {
	return css.Scope(key, rawCSS)
}

func UseStyles(ctx *Ctx, rawCSS string) *ScopedStyles {
	scoped := runtime.UseCell(ctx, func() *ScopedStyles { return ScopeCSS(ctx.ComponentName()+":"+rawCSS, rawCSS) })
	UseHead(ctx, Head{Styles: []StyleTag{{ID: scoped.Hash, CSS: scoped.Text}}})
	return scoped
}

func UseRouter(ctx *Ctx) *Router {
	return router.UseRouter(ctx)
}

func UseRequest(ctx *Ctx) *Request {
	return headers.UseRequest(ctx)
}

func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	return runtime.UseHandler(ctx, method, chain...)
}

func UseUpload(ctx *Ctx) *UploadHandle {
	return runtime.UseUpload(ctx)
}

func UseHead(ctx *Ctx, h Head) {
	head.UseHead(ctx, h)
}

func UseErrorBoundary(ctx *Ctx) *Boundary {
	return runtime.UseErrorBoundary(ctx)
}

func Status(code int) NavOption { return headers.Status(code) }

func RenderHTML(n Node) string { return runtime.RenderHTML(n) }
