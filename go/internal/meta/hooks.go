package meta

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// UseMetaTags sets document metadata (title, meta tags, links, scripts) for the current page.
// Useful for SEO, Open Graph tags, and dynamic page titles in single-page applications.
// Meta from deeper components (children) takes priority over shallower components (parents).
// Meta tags with the same name/property are merged, with child values overriding parent values.
func UseMetaTags(ctx runtime.Ctx, meta *Meta) {
	controller := metaCtx.Use(ctx)
	if controller == nil {
		return
	}

	componentID := ctx.ComponentID()
	depth := ctx.ComponentDepth()

	controller.Set(componentID, depth, meta)

	runtime.UseEffect(ctx, func() runtime.Cleanup {
		return func() {
			controller.Remove(componentID)
		}
	})
}
