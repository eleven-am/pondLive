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
	entriesRef := state.entriesRef

	runtime.UseEffect(ctx, func() func() {
		var current map[string]documentEntry
		if entriesRef != nil {
			current = entriesRef.Current
		} else {
			current = state.entries
		}

		next := maps.Clone(current)
		next[componentID] = documentEntry{doc: doc, depth: depth, componentID: componentID}
		state.setEntries(next)

		return func() {
			var cleanupCurrent map[string]documentEntry
			if entriesRef != nil {
				cleanupCurrent = entriesRef.Current
			} else {
				cleanupCurrent = state.entries
			}

			cleaned := maps.Clone(cleanupCurrent)
			delete(cleaned, componentID)
			state.setEntries(cleaned)
		}
	}, doc)
}
