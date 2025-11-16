package router

import (
	h "github.com/eleven-am/pondlive/go/internal/html"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

func RouterRedirect(ctx runtime.Ctx, to string) h.Node {
	state := requireRouterState(ctx)
	target := resolveHref(state.getLoc(), to)
	href := BuildHref(target.Path, target.Query, target.Hash)
	runtime.UseEffect(ctx, func() runtime.Cleanup {
		performLocationUpdate(ctx, target, true, true)
		return nil
	}, href)
	return h.Fragment()
}
