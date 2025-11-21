package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func Redirect(ctx runtime.Ctx, to string) *dom.StructuredNode {
	controller := UseRouterState(ctx)
	state := controller.Get()
	target := resolveHref(state.Location, to)
	href := BuildHref(target.Path, target.Query, target.Hash)
	runtime.UseEffect(ctx, func() runtime.Cleanup {
		performLocationUpdate(ctx, target, true, true)
		return nil
	}, href)
	return dom.FragmentNode()
}
