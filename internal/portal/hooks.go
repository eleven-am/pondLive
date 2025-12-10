package portal

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func UsePortal(ctx *runtime.Ctx, items []work.Item) {
	state := portalCtx.UseContextValue(ctx)
	if state == nil {
		return
	}

	nodes := work.ItemsToNodes(items)
	state.nodes = append(state.nodes, nodes...)
}
