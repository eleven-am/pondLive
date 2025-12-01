package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/router"
)

type (
	Location      = router.Location
	MatchState    = router.MatchState
	Match         = router.Match
	RouteProps    = router.RouteProps
	RoutesProps   = router.RoutesProps
	LinkProps     = router.LinkProps
	NavLinkProps  = router.NavLinkProps
	RedirectProps = router.RedirectProps
	OutletProps   = router.OutletProps
)

func Navigate(ctx *Ctx, href string) {
	router.Navigate(ctx, href)
}

func Replace(ctx *Ctx, href string) {
	router.Replace(ctx, href)
}

func Back(ctx *Ctx) {
	router.Back(ctx)
}

func Forward(ctx *Ctx) {
	router.Forward(ctx)
}

func UseLocation(ctx *Ctx) *Location {
	return router.UseLocation(ctx)
}

func UseParams(ctx *Ctx) map[string]string {
	return router.UseParams(ctx)
}

func UseRouteParam(ctx *Ctx, key string) string {
	return router.UseParam(ctx, key)
}

func UseMatch(ctx *Ctx) *MatchState {
	return router.UseMatch(ctx)
}

func UseMatched(ctx *Ctx) bool {
	return router.UseMatched(ctx)
}

func Link(ctx *Ctx, props LinkProps, children ...Item) Node {
	return router.Link(ctx, props, children...)
}

func NavLink(ctx *Ctx, props NavLinkProps, children ...Item) Node {
	return router.NavLink(ctx, props, children...)
}

func Route(ctx *Ctx, props RouteProps, children ...Node) Node {
	return router.Route(ctx, props, children...)
}

func Routes(ctx *Ctx, props RoutesProps, children ...Item) Node {
	return router.Routes(ctx, props, children...)
}

func Redirect(ctx *Ctx, props RedirectProps) Node {
	return router.Redirect(ctx, props)
}

func RedirectIf(ctx *Ctx, condition bool, to string, otherwise Node) Node {
	return router.RedirectIf(ctx, condition, to, otherwise)
}

func RedirectIfNot(ctx *Ctx, condition bool, to string, otherwise Node) Node {
	return router.RedirectIfNot(ctx, condition, to, otherwise)
}

func Outlet(ctx *Ctx, props OutletProps, fallback ...Item) Node {
	return router.Outlet(ctx, props, fallback...)
}

func OutletDefault(ctx *Ctx, fallback ...Item) Node {
	return router.OutletDefault(ctx, fallback...)
}

func HasOutlet(ctx *Ctx, name string) bool {
	return router.HasOutlet(ctx, name)
}

func OutletNames(ctx *Ctx) []string {
	return router.OutletNames(ctx)
}
