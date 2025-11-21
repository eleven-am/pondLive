package meta

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// UseMetaTags sets document metadata (title, meta tags, links, scripts) for the current page.
// Useful for SEO, Open Graph tags, and dynamic page titles in single-page applications.
func UseMetaTags(ctx runtime.Ctx, meta *Meta) {
	controller := metaCtx.Use(ctx)
	if controller != nil {
		controller.Set(meta)
	}
}
