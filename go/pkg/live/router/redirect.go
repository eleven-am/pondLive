package router

import (
	ui "github.com/eleven-am/go/pondlive/pkg/live"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

func Redirect(ctx ui.Ctx, to string) ui.Node {
	state := requireRouterState(ctx)
	target := resolveHref(state.getLoc(), to)
	href := BuildHref(target.Path, target.Query, target.Hash)
	ui.UseEffect(ctx, func() ui.Cleanup {
		performLocationUpdate(ctx, target, true, true)
		return nil
	}, href)
	return h.Fragment()
}
