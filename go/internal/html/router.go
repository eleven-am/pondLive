package html

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

func Link(ctx *Ctx, props LinkProps, children []Node) Node {
	return router.Link(ctx, props, children)
}

func NavLink(ctx *Ctx, props NavLinkProps, children []Node) Node {
	return router.NavLink(ctx, props, children)
}

func Route(ctx *Ctx, props RouteProps) Node {
	return router.Route(ctx, props)
}

func Routes(ctx *Ctx, props RoutesProps, children []Node) Node {
	return router.Routes(ctx, props, children)
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
