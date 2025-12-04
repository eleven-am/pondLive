package document

import (
	"maps"

	"github.com/eleven-am/pondlive/internal/runtime"
)

func UseDocument(ctx *runtime.Ctx, doc *Document) {
	state := documentCtx.UseContextValue(ctx)
	if state == nil {
		return
	}

	componentID := ctx.ComponentID()
	depth := ctx.ComponentDepth()

	runtime.UseEffect(ctx, func() func() {
		next := maps.Clone(state.entries)
		next[componentID] = documentEntry{doc: doc, depth: depth, componentID: componentID}
		state.setEntries(next)

		return func() {
			cleaned := maps.Clone(state.entries)
			delete(cleaned, componentID)
			state.setEntries(cleaned)
		}
	}, doc)
}
