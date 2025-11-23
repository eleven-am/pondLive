package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Controller provides access to router state.
// Location is read from RequestController (single source of truth).
// Match state is owned by the router.
type Controller struct {
	requestController *headers.RequestController
	getMatch          func() *MatchState
	setMatch          func(*MatchState)
}

// NewController creates a new router controller.
// Requires a RequestController to read location from.
func NewController(
	requestController *headers.RequestController,
	getMatch func() *MatchState,
	setMatch func(*MatchState),
) *Controller {
	return &Controller{
		requestController: requestController,
		getMatch:          getMatch,
		setMatch:          setMatch,
	}
}

// GetLocation reads the current location from RequestController.
// This is the single source of truth for location state.
func (c *Controller) GetLocation() Location {
	if c == nil || c.requestController == nil {
		return Location{Path: "/"}
	}

	path, query, hash := c.requestController.GetCurrentLocation()
	return Location{
		Path:  path,
		Query: query,
		Hash:  hash,
	}
}

// GetMatch returns the current match state.
func (c *Controller) GetMatch() *MatchState {
	if c == nil || c.getMatch == nil {
		return &MatchState{
			Matched: false,
			Pattern: "",
			Params:  make(map[string]string),
			Path:    "",
		}
	}
	return c.getMatch()
}

// SetMatch updates the match state.
// Called by Routes() when a route matches.
func (c *Controller) SetMatch(pattern string, params map[string]string, path string) {
	if c == nil || c.setMatch == nil {
		return
	}

	current := c.GetMatch()

	if current.Matched && current.Pattern == pattern && current.Path == path {
		return
	}

	c.setMatch(&MatchState{
		Matched: true,
		Pattern: pattern,
		Params:  params,
		Path:    path,
	})
}

// ClearMatch resets the match state.
// Called when location changes but no route matches.
func (c *Controller) ClearMatch() {
	if c == nil || c.setMatch == nil {
		return
	}

	c.setMatch(&MatchState{
		Matched: false,
		Pattern: "",
		Params:  make(map[string]string),
		Path:    "",
	})
}

// routerCtx is the context for providing router controller to child components.
var routerCtx = runtime.CreateContext[*Controller](nil)

// useRouterController returns the router controller from context.
// Internal only - users should use UseLocation, UseParams, etc.
func useRouterController(ctx Ctx) *Controller {
	return routerCtx.Use(ctx)
}

// ProvideRouterController provides the router controller to child components.
func ProvideRouterController(
	ctx Ctx,
	controller *Controller,
	render func(Ctx) *dom.StructuredNode,
) *dom.StructuredNode {
	return routerCtx.Provide(ctx, controller, render)
}

// UseLocation returns the current location.
// Reads from RequestController via the router controller.
func UseLocation(ctx Ctx) Location {
	controller := useRouterController(ctx)
	if controller == nil {
		return Location{Path: "/"}
	}
	return controller.GetLocation()
}

// UseParams returns the current route parameters.
// Returns a copy to prevent external mutation.
func UseParams(ctx Ctx) map[string]string {
	controller := useRouterController(ctx)
	if controller == nil {
		return map[string]string{}
	}

	match := controller.GetMatch()
	if !match.Matched || len(match.Params) == 0 {
		return map[string]string{}
	}

	params := make(map[string]string, len(match.Params))
	for k, v := range match.Params {
		params[k] = v
	}
	return params
}

// UseParam returns a single route parameter by key.
// Returns empty string if parameter doesn't exist.
//
// Usage:
//
//	id := router.UseParam(ctx, "id")  // from /users/:id
func UseParam(ctx Ctx, key string) string {
	if key == "" {
		return ""
	}
	params := UseParams(ctx)
	return params[key]
}

// UseQuery returns the current query parameters.
// Returns a copy to prevent external mutation.
//
// Usage:
//
//	query := router.UseQuery(ctx)
//	page := query.Get("page")
func UseQuery(ctx Ctx) url.Values {
	loc := UseLocation(ctx)
	return cloneValues(loc.Query)
}

// UseSearchParam returns getter and setter functions for a specific query parameter.
// The setter updates the query parameter and triggers navigation (replace mode).
//
// Usage:
//
//	getPage, setPage := router.UseSearchParam(ctx, "page")
//	currentPage := getPage()  // []string{"1"} or nil
//	setPage([]string{"2"})    // Updates ?page=2
func UseSearchParam(ctx Ctx, key string) (func() []string, func([]string)) {
	controller := useRouterController(ctx)

	get := func() []string {
		if controller == nil {
			return nil
		}
		loc := controller.GetLocation()
		values := loc.Query[key]
		if len(values) == 0 {
			return nil
		}

		out := make([]string, len(values))
		copy(out, values)
		return out
	}

	set := func(values []string) {
		if controller == nil || controller.requestController == nil {
			return
		}

		current := controller.GetLocation()

		next := cloneLocation(current)
		if next.Query == nil {
			next.Query = url.Values{}
		}

		if len(values) == 0 {
			delete(next.Query, key)
		} else {
			next.Query[key] = append([]string(nil), values...)
		}

		next = canonicalizeLocation(next)
		if locationEqual(current, next) {
			return
		}

		controller.requestController.SetCurrentLocation(next.Path, next.Query, next.Hash)
		recordNavigation(ctx, next, true)
	}

	return get, set
}

// UseSearchParams returns getter and setter functions for all query parameters.
// The setter allows batch updates to multiple query parameters at once.
//
// Usage:
//
//	getParams, setParams := router.UseSearchParams(ctx)
//	params := getParams()    // url.Values with all query params
//	page := params.Get("page")
//
//	// Update multiple params at once
//	setParams(url.Values{
//	    "page": []string{"2"},
//	    "sort": []string{"name"},
//	})
func UseSearchParams(ctx Ctx) (func() url.Values, func(url.Values)) {
	controller := useRouterController(ctx)

	get := func() url.Values {
		if controller == nil {
			return url.Values{}
		}
		loc := controller.GetLocation()
		return cloneValues(loc.Query)
	}

	set := func(newParams url.Values) {
		if controller == nil || controller.requestController == nil {
			return
		}

		current := controller.GetLocation()
		next := Location{
			Path:  current.Path,
			Query: cloneValues(newParams),
			Hash:  current.Hash,
		}

		next = canonicalizeLocation(next)
		if locationEqual(current, next) {
			return
		}

		controller.requestController.SetCurrentLocation(next.Path, next.Query, next.Hash)
		recordNavigation(ctx, next, true)
	}

	return get, set
}
