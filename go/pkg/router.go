package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/router"
)

type (
	Location      = router.Location
	MatchState    = router.MatchState
	Match         = router.Match
	RouteProps    = router.RouteProps
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

func UseLocation(ctx *Ctx) Location {
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

func Route(ctx *Ctx, props RouteProps, children ...Item) Node {
	nodes := make([]Node, 0, len(children))
	for _, item := range children {
		if node, ok := item.(Node); ok {
			nodes = append(nodes, node)
		}
	}
	return router.Route(ctx, props, nodes...)
}

func Routes(ctx *Ctx, children ...Item) Node {
	return router.Routes(ctx, children...)
}

func Redirect(ctx *Ctx, props RedirectProps) Node {
	return router.Redirect(ctx, props)
}

func Outlet(ctx *Ctx) Node {
	return router.Outlet(ctx)
}
