package router

import (
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Redirect performs a redirect to the specified href.
// During SSR, sets HTTP redirect headers.
// During live session, navigates client-side using replace mode.
//
// Usage:
//
//	func LoginRequired(ctx Ctx, match Match) *dom.StructuredNode {
//	    user := UseUser(ctx)
//	    if user == nil {
//	        return router.Redirect(ctx, "/login")
//	    }
//	    return DashboardPage(ctx)
//	}
func Redirect(ctx Ctx, to string) *dom.StructuredNode {
	controller := useRouterController(ctx)
	requestController := headers.UseRequestController(ctx)

	var href string
	if controller != nil {
		current := controller.GetLocation()
		target := resolveHref(current, to)
		target = canonicalizeLocation(target)
		href = buildHref(target.Path, target.Query, target.Hash)

		runtime.UseEffect(ctx, func() runtime.Cleanup {
			if ctx.IsLive() {
				controller.requestController.SetCurrentLocation(target.Path, target.Query, target.Hash)
				recordNavigation(ctx, target, true)
			}
			return nil
		}, href)
	} else {
		href = to
	}

	if !ctx.IsLive() {
		if requestController != nil {
			requestController.SetRedirect(href, http.StatusFound)
		}
	}

	return dom.FragmentNode()
}
