package runtime

import (
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func RouterRedirect(ctx Ctx, to string) h.Node {
	state := requireRouterState(ctx)
	target := resolveHref(state.getLoc(), to)
	href := BuildHref(target.Path, target.Query, target.Hash)
	UseEffect(ctx, func() Cleanup {
		performLocationUpdate(ctx, target, true, true)
		return nil
	}, href)
	return h.Fragment()
}
