package live

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/route"
	"github.com/eleven-am/pondlive/go/internal/router"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type (
	Location    = router.Location
	RouteProps  = router.RouteProps
	RoutesProps = router.RoutesProps
	LinkProps   = router.LinkProps
	Match       = router.Match
)

var (
	Parse            = route.Parse
	NormalizePattern = route.NormalizePattern
	Prefer           = route.Prefer
	BestMatch        = route.BestMatch
	BuildHref        = route.BuildHref
	SetSearch        = route.SetSearch
	AddSearch        = route.AddSearch
	DelSearch        = route.DelSearch
	MergeSearch      = route.MergeSearch
	ClearSearch      = route.ClearSearch
	ParseHref        = route.ParseHref
	ErrMissingRouter = route.ErrMissingRouter
	LocEqual         = route.LocEqual
)

func Router(ctx Ctx, children ...Node) Node {
	return router.Router(ctx, children...)
}

func Routes(ctx Ctx, props router.RoutesProps, children ...Node) Node {
	return router.Routes(ctx, props, children...)
}

func Route(ctx Ctx, props RouteProps, children ...Node) Node {
	return router.Route(ctx, props, children...)
}

func Outlet(ctx Ctx, name ...string) Node {
	return router.Outlet(ctx, name...)
}

func Link(ctx Ctx, props LinkProps, children ...h.Item) Node {
	return router.Link(ctx, props, h.Fragment(children...))
}

func Navigate(ctx Ctx, href string) {
	router.Navigate(ctx, href)
}

func Replace(ctx Ctx, href string) {
	router.Replace(ctx, href)
}

func NavigateWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	router.NavigateWithSearch(ctx, patch)
}

func ReplaceWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	router.ReplaceWithSearch(ctx, patch)
}

func Redirect(ctx Ctx, to string) Node {
	return router.Redirect(ctx, to)
}

// UseLocation returns the current location (path, query, hash).
func UseLocation(ctx Ctx) Location {
	return router.UseLocation(ctx)
}

// UseParams returns all route parameters from the URL pattern.
// Example: pattern "/users/:id" with URL "/users/123" returns {"id": "123"}
func UseParams(ctx Ctx) map[string]string {
	return router.UseParams(ctx)
}

// UseParam returns a single route parameter by key.
// Returns empty string if not found.
func UseParam(ctx Ctx, key string) string {
	return router.UseParam(ctx, key)
}

// UseSearch returns the current query parameters as url.Values.
func UseSearch(ctx Ctx) url.Values {
	return router.UseQuery(ctx)
}

// UseSearchParam returns getter/setter for a specific query parameter.
// Setting the value updates the URL and triggers a render.
func UseSearchParam(ctx Ctx, key string) (func() []string, func([]string)) {
	return router.UseSearchParam(ctx, key)
}
