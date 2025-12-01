package metatags

import (
	"maps"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func UseMetaTags(ctx *runtime.Ctx, meta *Meta) {
	state := metaCtx.UseContextValue(ctx)
	if state == nil {
		return
	}

	componentID := ctx.ComponentID()
	depth := ctx.ComponentDepth()

	runtime.UseEffect(ctx, func() func() {
		next := maps.Clone(state.entries)
		next[componentID] = metaEntry{meta: meta, depth: depth, componentID: componentID}
		state.setEntries(next)

		return func() {
			currentState := metaCtx.UseContextValue(ctx)
			if currentState == nil {
				return
			}
			cleaned := maps.Clone(currentState.entries)
			delete(cleaned, componentID)
			currentState.setEntries(cleaned)
		}
	}, meta)
}
