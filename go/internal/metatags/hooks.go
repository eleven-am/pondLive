package metatags

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func UseMetaTags(ctx *runtime.Ctx, meta *Meta) {
	controller := metaCtx.UseContextValue(ctx)
	if controller == nil {
		return
	}

	componentID := ctx.ComponentID()
	depth := ctx.ComponentDepth()

	controller.Set(componentID, depth, meta)

	runtime.UseEffect(ctx, func() func() {
		return func() {
			controller.Remove(componentID)
		}
	})
}
