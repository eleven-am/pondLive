package metatags

import (
	"github.com/eleven-am/pondlive/internal/runtime"
)

func UseMetaTags(ctx *runtime.Ctx, meta *Meta) {
	funcs := providerCtx.UseContextValue(ctx)
	if funcs == nil {
		return
	}

	componentID := ctx.ComponentID()
	depth := ctx.ComponentDepth()

	runtime.UseEffect(ctx, func() func() {
		funcs.update(componentID, metaEntry{meta: meta, depth: depth, componentID: componentID})

		return func() {
			funcs.remove(componentID)
		}
	}, meta)
}
