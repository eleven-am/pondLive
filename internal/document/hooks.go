package document

import (
	"maps"

	"github.com/eleven-am/pondlive/internal/runtime"
)

func UseDocument(ctx *runtime.Ctx) *DocumentHandle {
	state := documentCtx.UseContextValue(ctx)
	if state == nil {
		return &DocumentHandle{}
	}

	handle := newDocumentHandle(ctx, state)

	runtime.UseEffect(ctx, func() func() {
		return func() {
			cleaned := maps.Clone(state.entries)
			delete(cleaned, handle.componentID)
			state.setEntries(cleaned)
		}
	})

	return handle
}
