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
			cleaned := maps.Clone(state.entries)
			delete(cleaned, componentID)
			state.setEntries(cleaned)
		}
	}, meta)
}
