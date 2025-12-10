package portal

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var Portal = runtime.Component(func(ctx *runtime.Ctx, items []work.Item) work.Node {
	state := portalCtx.UseContextValue(ctx)
	if state == nil {
		return work.NewFragment()
	}

	nodes := work.ItemsToNodes(items)
	state.nodes = append(state.nodes, nodes...)

	return work.NewFragment()
})
