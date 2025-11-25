package meta2

import (
	"github.com/eleven-am/pondlive/go/internal/runtime2"
)

// UseMetaTags sets document metadata (title, meta tags, links, scripts) for the current page.
// Useful for SEO, Open Graph tags, and dynamic page titles in single-page applications.
// Meta from deeper components (children) takes priority over shallower components (parents).
// Meta tags with the same name/property are merged, with child values overriding parent values.
func UseMetaTags(ctx *runtime2.Ctx, meta *Meta) {
	controller := metaCtx.UseContextValue(ctx)
	if controller == nil {
		return
	}

	componentID := ctx.ComponentID()
	depth := ctx.ComponentDepth()

	controller.Set(componentID, depth, meta)

	runtime2.UseEffect(ctx, func() func() {
		return func() {
			controller.Remove(componentID)
		}
	})
}
