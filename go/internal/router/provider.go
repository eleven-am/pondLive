package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// ProvideRouter sets up the router system and provides it to the application tree.
// This is the main entry point for router.
//
// Key responsibilities:
// 1. Gets RequestController from context (source of truth for location)
// 2. Creates router controller that reads location from RequestController
// 3. Provides router controller to tree
// 4. Sets up slot context for outlets
//
// Usage:
//
//	func App(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
//	    return router.ProvideRouter(ctx, func(rctx router.Ctx) *dom.StructuredNode {
//	        return h.Div(
//	            router.Routes(rctx, router.RoutesProps{Outlet: "main"},
//	                router.Route(rctx, router.RouteProps{Path: "/", Component: HomePage}),
//	            ),
//	            h.Main(router.Outlet(rctx)),
//	        )
//	    })
//	}
func ProvideRouter(ctx Ctx, render func(Ctx) *dom.StructuredNode) *dom.StructuredNode {

	requestController := headers.UseRequestController(ctx)
	if requestController == nil {

		requestController = headers.NewRequestController()
		requestController.SetInitialLocation("/", nil, "")
	}

	matchState, setMatchState := runtime.UseState(ctx, &MatchState{
		Matched: false,
		Pattern: "",
		Params:  make(map[string]string),
		Path:    "",
	})

	controller := runtime.UseMemo(ctx, func() *Controller {
		return NewController(requestController, matchState, setMatchState)
	})

	return ProvideRouterController(ctx, controller, func(rctx Ctx) *dom.StructuredNode {

		children := render(rctx)

		return routerOutletSlotCtx.Provide(rctx, []dom.Item{children}, func(sctx runtime.Ctx) *dom.StructuredNode {

			return children
		})
	})
}
