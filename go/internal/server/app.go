package server

import (
	"context"
	"net/http"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

// App is the main application server with explicit routing.
type App struct {
	mux         *http.ServeMux
	ctx         context.Context
	pondManager *pond.Manager
}

// NewApp creates a new server application.
func NewApp(ctx context.Context) *App {
	if ctx == nil {
		ctx = context.Background()
	}

	app := &App{
		mux:         http.NewServeMux(),
		ctx:         ctx,
		pondManager: pond.NewManager(ctx),
	}

	return app
}

// Handler returns the http.Handler for this app.
func (a *App) Handler() http.Handler {
	return a.mux
}

// Mux returns the underlying ServeMux for advanced use cases.
func (a *App) Mux() *http.ServeMux {
	return a.mux
}

// PondManager returns the PondSocket manager for registering endpoints.
func (a *App) PondManager() *pond.Manager {
	return a.pondManager
}

// HandleFunc registers a handler function for the given pattern.
func (a *App) HandleFunc(pattern string, handler http.HandlerFunc) {
	a.mux.HandleFunc(pattern, handler)
}

// Handle registers a handler for the given pattern.
func (a *App) Handle(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

// Group creates a middleware chain that can be applied to multiple routes.
type Group struct {
	app        *App
	prefix     string
	middleware []func(http.Handler) http.Handler
}

// NewGroup creates a new route group with optional prefix and middleware.
func (a *App) NewGroup(prefix string, middleware ...func(http.Handler) http.Handler) *Group {
	return &Group{
		app:        a,
		prefix:     prefix,
		middleware: middleware,
	}
}

// HandleFunc registers a handler in this group with middleware applied.
func (g *Group) HandleFunc(pattern string, handler http.HandlerFunc) {
	fullPattern := g.prefix + pattern

	h := http.Handler(handler)
	for i := len(g.middleware) - 1; i >= 0; i-- {
		h = g.middleware[i](h)
	}

	g.app.mux.Handle(fullPattern, h)
}

// Handle registers a handler in this group with middleware applied.
func (g *Group) Handle(pattern string, handler http.Handler) {
	fullPattern := g.prefix + pattern

	h := handler
	for i := len(g.middleware) - 1; i >= 0; i-- {
		h = g.middleware[i](h)
	}

	g.app.mux.Handle(fullPattern, h)
}
