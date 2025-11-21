package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// RouterState holds all router state in a single consolidated structure.
type State struct {
	Location Location          // Current location (path, query, hash)
	Matched  bool              // Has a route matched?
	Pattern  string            // Matched route pattern
	Params   map[string]string // Route parameters extracted from match
	Path     string            // Matched path
}

// Controller provides get/set access to router state.
type Controller struct {
	get func() *State
	set func(*State)
}

// NewController creates a new router state controller with the given get/set functions.
func NewController(get func() *State, set func(*State)) *Controller {
	return &Controller{
		get: get,
		set: set,
	}
}

// Get returns the current router state.
func (c *Controller) Get() *State {
	if c == nil || c.get == nil {
		return defaultRouterState
	}
	return c.get()
}

// Set updates the router state.
func (c *Controller) Set(state *State) {
	if c != nil && c.set != nil {
		c.set(state)
	}
}

// SetLocation updates only the location, resetting match state.
func (c *Controller) SetLocation(loc Location) {
	if c == nil {
		return
	}
	c.Set(&State{
		Location: loc,
		Matched:  false,
		Pattern:  "",
		Params:   nil,
		Path:     "",
	})
}

// SetMatch marks a route as matched and stores its details.
// If the route is already matched with the same pattern and path, this is a no-op
// to prevent unnecessary re-renders.
func (c *Controller) SetMatch(pattern string, params map[string]string, path string) {
	if c == nil {
		return
	}
	state := c.Get()

	if state.Matched && state.Pattern == pattern && state.Path == path {
		return
	}

	c.Set(&State{
		Location: state.Location,
		Matched:  true,
		Pattern:  pattern,
		Params:   params,
		Path:     path,
	})
}

var defaultRouterState = &State{
	Location: Location{
		Path:  "/",
		Query: url.Values{},
		Hash:  "",
	},
	Matched: false,
	Pattern: "",
	Params:  make(map[string]string),
	Path:    "",
}

// routerCtx is the context for providing router state controller to child components.
var routerCtx = runtime.CreateContext[*Controller](&Controller{
	get: func() *State { return defaultRouterState },
	set: func(*State) {},
})

// UseRouterState returns the router state controller from context.
func UseRouterState(ctx runtime.Ctx) *Controller {
	return routerCtx.Use(ctx)
}

// ProvideRouterState provides the router state controller to child components.
func ProvideRouterState(ctx runtime.Ctx, controller *Controller, render func(runtime.Ctx) *dom.StructuredNode) *dom.StructuredNode {
	return routerCtx.Provide(ctx, controller, render)
}
