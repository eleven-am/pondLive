package portal

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var Target = runtime.Component(func(ctx *runtime.Ctx, children []work.Item) work.Node {
	state := portalCtx.UseContextValue(ctx)
	if state == nil || len(state.nodes) == 0 {
		return nil
	}

	return &work.Fragment{Children: state.nodes}
})
