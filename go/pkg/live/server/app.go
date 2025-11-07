package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	livehttp "github.com/eleven-am/pondlive/go/internal/server/http"
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
	version        int
	idGenerator    func(*http.Request) (runtime.SessionID, error)
	session        *runtime.LiveSessionConfig
	clientAssetURL string
	devMode        *bool
}

// WithVersion overrides the server protocol version advertised in init and frame payloads.
func WithVersion(v int) Option {
	return func(cfg *appOptions) {
		if cfg != nil {
			cfg.version = v
		}
	}
}

// WithIDGenerator replaces the default session ID allocator.
func WithIDGenerator(fn func(*http.Request) (runtime.SessionID, error)) Option {
	return func(cfg *appOptions) {
		if cfg != nil {
			cfg.idGenerator = fn
		}
	}
}

// WithSessionConfig applies runtime session settings such as TTL or frame history.
func WithSessionConfig(cfg runtime.LiveSessionConfig) Option {
	return func(opts *appOptions) {
		if opts == nil {
			return
		}
		clone := cfg
		opts.session = &clone
	}
}

// WithClientAssetURL customizes the URL used when serving the embedded client bundle.
func WithClientAssetURL(url string) Option {
	return func(cfg *appOptions) {
		if cfg != nil {
			cfg.clientAssetURL = url
		}
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

// NewApp constructs a LiveUI application stack using the supplied component.
// The returned handler automatically serves the embedded client script, PondSocket
// endpoint, upload handler, and cookie negotiation endpoint used by SetCookie
// and DeleteCookie, so applications do not need to register those routes
// manually.
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

	managerCfg := &livehttp.ManagerConfig{
		Component:      adaptComponent(component),
		ClientAssetURL: clientScriptPath(devMode),
	}
	if applied.version > 0 {
		managerCfg.Version = applied.version
	}
	if applied.idGenerator != nil {
		managerCfg.IDGenerator = applied.idGenerator
	}
	var sessionCfg *runtime.LiveSessionConfig
	if applied.session != nil {
		clone := *applied.session
		sessionCfg = &clone
	}
	if applied.devMode != nil {
		if sessionCfg == nil {
			sessionCfg = &runtime.LiveSessionConfig{}
		}
		value := *applied.devMode
		sessionCfg.DevMode = &value
	}
	if sessionCfg != nil {
		managerCfg.Session = sessionCfg
	}
	if strings.TrimSpace(applied.clientAssetURL) != "" {
		managerCfg.ClientAssetURL = applied.clientAssetURL
	}

	manager := livehttp.NewManager(managerCfg)

	pondManager := pond.NewManager(ctx)
	pondEndpoint, err := manager.RegisterPondSocket(pondManager)
	if err != nil {
		return nil, err
	}

	prefix := trimPatternPrefix(livehttp.PondSocketPattern)
	route := endpointFromPrefix(prefix)
	manager.SetClientConfig(protocol.ClientConfig{Endpoint: route, UploadEndpoint: livehttp.UploadPathPrefix})

	scriptPath := managerCfg.ClientAssetURL
	if strings.TrimSpace(scriptPath) == "" {
		scriptPath = clientScriptPath(devMode)
	}
	scriptHandler := embeddedClientScriptHandler(scriptPath)
	sourceMapPath := ""
	sourceMapHandler := http.Handler(nil)
	if devMode {
		sourceMapPath = clientSourceMapPath(scriptPath)
		sourceMapHandler = embeddedClientSourceMapHandler(sourceMapPath)
	}

	mux := http.NewServeMux()
	mux.Handle(route, pondManager.HTTPHandler())
	mux.Handle(scriptPath, scriptHandler)
	if devMode && sourceMapHandler != nil && strings.TrimSpace(sourceMapPath) != "" {
		mux.Handle(sourceMapPath, sourceMapHandler)
	}
	mux.Handle(livehttp.UploadPathPrefix, livehttp.NewUploadHandler(manager.Registry()))
	mux.Handle(livehttp.CookiePath, livehttp.NewCookieHandler(manager.Registry()))
	mux.Handle("/", manager)

	provider := pondEndpoint.PubsubProvider()
	if provider != nil {
		adapter := runtime.WrapPubsubProvider(provider)
		runtime.SetDefaultPubsubPublisher(runtime.PubsubPublishFunc(adapter))
	} else {
		runtime.SetDefaultPubsubPublisher(nil)
	}

	return &App{handler: mux, provider: provider}, nil
}

func adaptComponent(fn Component) runtime.Component[struct{}] {
	if fn == nil {
		return nil
	}
	return func(ctx runtime.Ctx, _ struct{}) h.Node {
		return fn(ctx)
	}
}

func trimPatternPrefix(pattern string) string {
	if pattern == "" {
		return "/"
	}
	cutoff := len(pattern)
	for _, marker := range []string{":", "*"} {
		if idx := strings.Index(pattern, marker); idx >= 0 && idx < cutoff {
			cutoff = idx
		}
	}
	prefix := pattern[:cutoff]
	if prefix == "" {
		prefix = "/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

func endpointFromPrefix(prefix string) string {
	if prefix == "" {
		return "/"
	}
	trimmed := strings.TrimSuffix(prefix, "/")
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

// Handler exposes the combined HTTP handler serving both SSR and websocket traffic.
func (a *App) Handler() http.Handler {
	if a == nil {
		return nil
	}
	return a.handler
}
