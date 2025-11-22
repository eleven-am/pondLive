package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/go/internal/handler"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/session"
	ui "github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// Component represents the root LiveUI component rendered for every request.
// It receives the LiveUI context and returns the HTML body fragment for SSR.
type Component func(ui.Ctx) h.Node

// Option configures the behavior of NewApp without requiring callers to construct
// the underlying HTTP manager or runtime primitives directly.
type Option func(*appOptions)

type appOptions struct {
	idGenerator func(*http.Request) (session.SessionID, error)
	session     *session.Config
	devMode     *bool
}

// WithIDGenerator replaces the default session ID allocator.
func WithIDGenerator(fn func(*http.Request) (session.SessionID, error)) Option {
	return func(cfg *appOptions) {
		if cfg != nil {
			cfg.idGenerator = fn
		}
	}
}

// WithSessionConfig applies runtime session settings such as TTL or frame history.
func WithSessionConfig(cfg session.Config) Option {
	return func(opts *appOptions) {
		if opts == nil {
			return
		}
		clone := cfg
		opts.session = &clone
	}
}

// WithDevMode enables developer-centric runtime features such as diagnostic
// streaming, recovery affordances, and serving the debug client bundle when set
// to true.
func WithDevMode(enabled bool) Option {
	return func(cfg *appOptions) {
		if cfg != nil {
			value := enabled
			cfg.devMode = &value
		}
	}
}

// App wires a LiveUI HTTP manager with a PondSocket transport and exposes a single
// http.Handler that serves both SSR and websocket traffic.
type App struct {
	handler  http.Handler
	provider runtime.PubsubProvider
}

const (
	PondSocketPath = "/live"
	CookiePath     = "/__cookie"
	HandlerPath    = "/_handlers/"
)

// NewApp constructs a LiveUI application stack using the supplied component.
// The returned handler automatically serves the embedded client script, PondSocket
// endpoint, and cookie negotiation endpoint used by SetCookie and DeleteCookie,
// so applications do not need to register those routes manually.
func NewApp(ctx context.Context, component Component, opts ...Option) (*App, error) {
	if component == nil {
		return nil, errors.New("live: component is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	applied := appOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&applied)
		}
	}

	devMode := false
	if applied.devMode != nil {
		devMode = *applied.devMode
	}

	var sessionCfg *session.Config
	if applied.session != nil {
		clone := *applied.session
		sessionCfg = &clone
	}
	if applied.devMode != nil {
		if sessionCfg == nil {
			sessionCfg = &session.Config{}
		}
		sessionCfg.DevMode = *applied.devMode
	}

	clientAsset := clientScriptPath(devMode)

	pondManager := pond.NewManager(ctx)

	registry := server.NewSessionRegistry()
	endpoint, err := server.Register(pondManager, PondSocketPath, registry)
	if err != nil {
		return nil, err
	}

	pubsubProvider := endpoint.PubsubProvider()

	ssrHandler := server.NewSSRHandler(server.SSRConfig{
		Registry:       registry,
		Component:      adaptComponent(component),
		IDGenerator:    applied.idGenerator,
		SessionConfig:  sessionCfg,
		ClientAsset:    clientAsset,
		ClientConfig:   &protocol.ClientConfig{Endpoint: PondSocketPath},
		PubsubProvider: pubsubProvider,
	})

	mux := http.NewServeMux()

	scriptPath := clientAsset
	if strings.TrimSpace(scriptPath) == "" {
		scriptPath = clientScriptPath(devMode)
	}
	mux.Handle(scriptPath, embeddedClientScriptHandler(scriptPath))

	if devMode {
		sourceMapPath := clientSourceMapPath(scriptPath)
		if strings.TrimSpace(sourceMapPath) != "" {
			mux.Handle(sourceMapPath, embeddedClientSourceMapHandler(sourceMapPath))
		}
	}

	mux.Handle(PondSocketPath, pondManager.HTTPHandler())

	mux.Handle(HandlerPath, handler.NewDispatcher(registry))

	mux.Handle("/", ssrHandler)

	return &App{handler: mux, provider: pubsubProvider}, nil
}

func adaptComponent(fn Component) session.Component {
	if fn == nil {
		return nil
	}
	return func(ctx runtime.Ctx) h.Node {
		return fn(ctx)
	}
}

// Handler exposes the combined HTTP handler serving both SSR and websocket traffic.
func (a *App) Handler() http.Handler {
	if a == nil {
		return nil
	}
	return a.handler
}
