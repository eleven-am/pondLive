package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func Redirect(ctx runtime.Ctx, to string) *dom2.StructuredNode {
	state := requireRouterState(ctx)
	target := resolveHref(state.getLoc(), to)
	href := BuildHref(target.Path, target.Query, target.Hash)
	runtime.UseEffect(ctx, func() runtime.Cleanup {
		performLocationUpdate(ctx, target, true, true)
		return nil
	}, href)
	return dom2.FragmentNode()
}
