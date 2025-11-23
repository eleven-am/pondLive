package router

import "github.com/eleven-am/pondlive/go/internal/dom"

// Router wraps children in a fragment node.
// This is a legacy component provided for backward compatibility with router v1.
// In router, router state is provided by ProvideRouter, so this is just a convenience wrapper.
//
// Usage:
//
//	router.Router(ctx,
//	    router.Routes(ctx, router.RoutesProps{Outlet: "default"},
//	        router.Route(ctx, router.RouteProps{Path: "/", Component: HomePage}),
//	    ),
//	    router.Outlet(ctx),
//	)
func Router(ctx Ctx, children ...*dom.StructuredNode) *dom.StructuredNode {
	if len(children) == 0 {
		return dom.FragmentNode()
	}
	items := make([]dom.Item, 0, len(children))
	for _, child := range children {
		if child != nil {
			items = append(items, child)
		}
	}
	return dom.FragmentNode(items...)
}
