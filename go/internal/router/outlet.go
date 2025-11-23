package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/slot"
)

// routerOutletSlotCtx is the package-level slot context used for all outlets.
// Router provides matched route components as slot content.
// Outlets render by pulling from this slot context.
var routerOutletSlotCtx = slot.CreateSlotContext()

// Outlet renders the content for a named outlet.
// This is a pure wrapper around the slot system - no router-specific logic.
//
// Usage:
//
//	router.Outlet(ctx)              // Renders "default" outlet
//	router.Outlet(ctx, "sidebar")   // Renders "sidebar" outlet
//
// The outlet receives content from Routes() components that specified
// the same outlet name in their RoutesProps.
func Outlet(ctx Ctx, name ...string) *dom.StructuredNode {
	outletName := "default"
	if len(name) > 0 && name[0] != "" {
		outletName = name[0]
	}

	return routerOutletSlotCtx.Render(ctx, outletName)
}

// OutletFallback sets fallback content for an outlet.
// This content renders when no route matches for the outlet.
// This is a pure wrapper around the slot system's fallback mechanism.
//
// Usage:
//
//	router.OutletFallback(ctx, "sidebar", EmptySidebarComponent)
//
// The fallback component receives the current Match from the router controller.
// This allows fallbacks to know the current route context even if their outlet didn't match.
func OutletFallback(ctx Ctx, outlet string, component Component[Match]) *dom.StructuredNode {
	controller := useRouterController(ctx)

	match := Match{}
	if controller != nil {
		loc := controller.GetLocation()
		matchState := controller.GetMatch()

		match = Match{
			Pattern:  matchState.Pattern,
			Path:     loc.Path,
			Params:   matchState.Params,
			Query:    loc.Query,
			RawQuery: loc.Query.Encode(),
			Hash:     loc.Hash,
		}
	}

	node := component(ctx, match)

	routerOutletSlotCtx.SetFallback(ctx, outlet, node)

	return dom.FragmentNode()
}

// HasOutlet checks if an outlet has content (either from a route or fallback).
// Useful for conditional rendering based on outlet availability.
//
// Usage:
//
//	if router.HasOutlet(ctx, "sidebar") {
//	    return h.Div(router.Outlet(ctx, "sidebar"))
//	}
func HasOutlet(ctx Ctx, outlet string) bool {
	return routerOutletSlotCtx.Has(ctx, outlet)
}
