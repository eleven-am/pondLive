package pkg

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/internal/document"
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/metatags"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/styles"
)

type (
	Ctx                       = runtime.Ctx
	Ref[T any]                = runtime.Ref[T]
	StateOpt[T any]           = runtime.StateOpt[T]
	Error                     = runtime.Error
	ErrorCode                 = runtime.ErrorCode
	ErrorBatch                = runtime.ErrorBatch
	PondError                 = runtime.PondError
	ServerError               = protocol.ServerError
	StackFrame                = runtime.StackFrame
	FrameCategory             = runtime.FrameCategory
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
	PresenceInput[T any]      = runtime.PresenceInput[T]
	PresenceResult[T any]     = runtime.PresenceResult[T]
	PresenceItem[T any]       = runtime.PresenceItem[T]
	Meta                      = metatags.Meta
	CookieOptions             = headers.CookieOptions
	Document                  = document.Document
	EventOptions              = metadata.EventOptions
)

const (
	FrameUser     = runtime.FrameUser
	FramePondLive = runtime.FramePondLive
	FrameRuntime  = runtime.FrameRuntime

	ErrCodeRender        = runtime.ErrCodeRender
	ErrCodeMemo          = runtime.ErrCodeMemo
	ErrCodeEffect        = runtime.ErrCodeEffect
	ErrCodeEffectCleanup = runtime.ErrCodeEffectCleanup
	ErrCodeHandler       = runtime.ErrCodeHandler
	ErrCodeValidation    = runtime.ErrCodeValidation
	ErrCodeApp           = runtime.ErrCodeApp
	ErrCodeSession       = runtime.ErrCodeSession
	ErrCodeNetwork       = runtime.ErrCodeNetwork
	ErrCodeTimeout       = runtime.ErrCodeTimeout
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

func UseErrorBoundary(ctx *Ctx) *ErrorBatch {
	return runtime.UseErrorBoundary(ctx)
}

func UseScript(ctx *Ctx, script string) ScriptHandle {
	return runtime.UseScript(ctx, script)
}

func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	return runtime.UseHandler(ctx, method, chain...)
}

func UseSlots(ctx *Ctx, items []Item) *SlotRenderer {
	return runtime.UseSlots(ctx, items)
}

func UseScopedSlots[T any](ctx *Ctx, items []Item) *ScopedSlotRenderer[T] {
	return runtime.UseScopedSlots[T](ctx, items)
}

func UseUpload(ctx *Ctx) *UploadHandle {
	return runtime.UseUpload(ctx)
}

func UseStream[T any](ctx *Ctx, renderRow func(StreamItem[T]) Node, initial ...StreamItem[T]) (Node, StreamHandle[T]) {
	return runtime.UseStream(ctx, renderRow, initial...)
}

func UseStyles(ctx *Ctx, rawCSS string) *Styles {
	return styles.UseStyles(ctx, rawCSS)
}

func UseHeaders(ctx *Ctx) http.Header {
	return headers.UseHeaders(ctx)
}

func UseCookie(ctx *Ctx, name string) (string, func(value string, opts *CookieOptions)) {
	return headers.UseCookie(ctx, name)
}

func UseMetaTags(ctx *Ctx, meta *Meta) {
	metatags.UseMetaTags(ctx, meta)
}

func UseDocument(ctx *Ctx, doc *Document) {
	document.UseDocument(ctx, doc)
}

func UseHydrated(ctx *Ctx, fn func() func(), deps ...any) {
	isLive := headers.UseIsLive(ctx)
	wasLiveOnMount := runtime.UseRef(ctx, isLive)

	allDeps := append([]any{isLive}, deps...)

	log.Printf("[UseHydrated] isLive=%v wasLiveOnMount=%v", isLive, wasLiveOnMount.Current)

	runtime.UseEffect(ctx, func() func() {
		log.Printf("[UseHydrated effect] isLive=%v wasLiveOnMount=%v", isLive, wasLiveOnMount.Current)
		if !isLive {
			return nil
		}

		if !wasLiveOnMount.Current {
			return fn()
		}

		var cancelled atomic.Bool
		var cleanupFn atomic.Pointer[func()]

		timer := time.AfterFunc(50*time.Millisecond, func() {
			if cancelled.Load() {
				return
			}
			if cleanup := fn(); cleanup != nil {
				cleanupFn.Store(&cleanup)
			}
		})

		return func() {
			cancelled.Store(true)
			timer.Stop()
			if c := cleanupFn.Load(); c != nil {
				(*c)()
			}
		}
	}, allDeps...)
}

func UsePresence[T any](ctx *Ctx, in PresenceInput[T]) PresenceResult[T] {
	return runtime.UsePresence(ctx, in)
}

func Present[T any](value T, dur time.Duration) PresenceInput[T] {
	return runtime.Present(value, dur)
}

func PresentIf[T any](condition bool, value T, dur time.Duration) PresenceInput[T] {
	return runtime.PresentIf(condition, value, dur)
}

func PresentWhen(condition bool, dur time.Duration) PresenceInput[struct{}] {
	return runtime.PresentWhen(condition, dur)
}

func PresentList[T any](items []T, keyFn func(T) string, dur time.Duration) PresenceInput[T] {
	return runtime.PresentList(items, keyFn, dur)
}
