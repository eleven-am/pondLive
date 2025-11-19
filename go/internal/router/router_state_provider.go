package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// StateProvider describes functions used to read and update router location state.
type StateProvider struct {
	Get func() Location
	Set func(Location)
}

// ProvideState attaches router state to the context and renders children.
func ProvideState(ctx runtime.Ctx, provider StateProvider, render func(runtime.Ctx) *dom2.StructuredNode) *dom2.StructuredNode {
	internal := routerState{
		getLoc: func() Location { return Location{Path: "/"} },
		setLoc: func(Location) {},
	}
	if provider.Get != nil {
		internal.getLoc = provider.Get
	}
	if provider.Set != nil {
		internal.setLoc = provider.Set
	}
	return RouterStateCtx.Provide(ctx, internal, func(rctx runtime.Ctx) *dom2.StructuredNode {
		loc := internal.getLoc()
		return LocationCtx.Provide(rctx, loc, render)
	})
}
