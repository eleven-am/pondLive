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

// App wires a LiveUI HTTP manager with a PondSocket transport and exposes a single
// http.Handler that serves both SSR and websocket traffic.
type App struct {
	handler  http.Handler
	provider runtime.PubsubProvider
}

// NewApp constructs a LiveUI application stack using the supplied component.
func NewApp(ctx context.Context, component Component) (*App, error) {
	if component == nil {
		return nil, errors.New("live: component is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	manager := livehttp.NewManager(&livehttp.ManagerConfig{
		Component:      adaptComponent(component),
		ClientAssetURL: clientScriptPath(),
	})

	pondManager := pond.NewManager(ctx)
	pondEndpoint, err := manager.RegisterPondSocket(pondManager)
	if err != nil {
		return nil, err
	}

	prefix := trimPatternPrefix(livehttp.PondSocketPattern)
	route := endpointFromPrefix(prefix)
	manager.SetClientConfig(protocol.ClientConfig{Endpoint: route})

	mux := http.NewServeMux()
	mux.Handle(route, pondManager.HTTPHandler())
	mux.Handle(clientScriptPath(), clientScriptHandler())
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
