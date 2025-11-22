package router

import (
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func Redirect(ctx runtime.Ctx, to string) *dom.StructuredNode {
	controller := UseRouterState(ctx)
	requestController := headers.UseRequestController(ctx)

	state := controller.Get()
	target := resolveHref(state.Location, to)
	href := BuildHref(target.Path, target.Query, target.Hash)

	runtime.UseEffect(ctx, func() runtime.Cleanup {
		if ctx.IsLive() {
			performLocationUpdate(ctx, target, true)
		}
		return nil
	}, href)

	if !ctx.IsLive() {
		if requestController != nil {
			requestController.SetRedirect(href, http.StatusFound)
		}
	}

	return dom.FragmentNode()
}
