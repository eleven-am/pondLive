package pkg

import (
	"github.com/eleven-am/pondlive/internal/css"
	"github.com/eleven-am/pondlive/internal/head"
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/shared"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

type (
	Node         = work.Node
	Item         = work.Item
	Event        = work.Event
	EventOptions = metadata.EventOptions

	Ctx                       = runtime.Ctx
	ReadSignal[T any]         = runtime.ReadSignal[T]
	Signal[T any]             = runtime.Signal[T]
	SignalOpt[T any]          = runtime.SignalOpt[T]
	Context[T any]            = runtime.Context[T]
	Source[T any]             = runtime.Source[T]
	ShowOpt                   = runtime.ShowOpt
	ForOpt                    = runtime.ForOpt
	SwitchArg                 = runtime.SwitchArg
	RegionOpt                 = runtime.RegionOpt
	Feed[T any, K comparable] = runtime.Feed[T, K]

	Timer = runtime.Timer

	Shared[T any]                     = shared.Shared[T]
	SharedMember[T any]               = shared.Member[T]
	SharedFamily[K comparable, T any] = shared.SharedFamily[K, T]
	SharedOpt                         = shared.SharedOpt
	Broadcast[T any]                  = shared.Broadcast[T]
	BroadcastOpt                      = shared.BroadcastOpt

	Script       = runtime.ScriptHandle
	ScopedStyles = css.Scoped

	Boundary      = runtime.Boundary
	Error         = runtime.Error
	Warning       = runtime.Warning
	WarningCode   = runtime.WarningCode
	ErrorCode     = runtime.ErrorCode
	ErrorBatch    = runtime.ErrorBatch
	PondError     = runtime.PondError
	ServerError   = protocol.ServerError
	Diagnostic    = protocol.Diagnostic
	StackFrame    = runtime.StackFrame
	FrameCategory = runtime.FrameCategory

	HandlerFunc      = runtime.HandlerFunc
	HandlerHandle    = runtime.HandlerHandle
	UploadHandle     = runtime.UploadHandle
	UploadHookConfig = runtime.UploadConfig
	UploadEvent      = runtime.UploadEvent
	UploadProgress   = runtime.UploadProgress
	UploadFileInfo   = upload.FileInfo

	Request       = headers.Request
	CookieOptions = headers.CookieOptions
	NavOption     = headers.NavOption

	Head      = head.Head
	MetaTag   = head.MetaTag
	LinkTag   = head.LinkTag
	ScriptTag = head.ScriptTag
	StyleTag  = head.StyleTag

	Router        = router.Router
	Location      = router.Location
	MatchState    = router.MatchState
	RouteDef      = router.RouteDef
	LinkProps     = router.LinkProps
	NavLinkProps  = router.NavLinkProps
	RedirectProps = router.RedirectProps
)

const (
	FrameUser     = runtime.FrameUser
	FramePondLive = runtime.FramePondLive
	FrameRuntime  = runtime.FrameRuntime

	ErrCodeRender        = runtime.ErrCodeRender
	ErrCodeEffect        = runtime.ErrCodeEffect
	ErrCodeEffectCleanup = runtime.ErrCodeEffectCleanup
	ErrCodeHandler       = runtime.ErrCodeHandler
	ErrCodeValidation    = runtime.ErrCodeValidation
	ErrCodeApp           = runtime.ErrCodeApp
	ErrCodeSession       = runtime.ErrCodeSession
	ErrCodeNetwork       = runtime.ErrCodeNetwork
	ErrCodeTimeout       = runtime.ErrCodeTimeout
)
