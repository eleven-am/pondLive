package live

import (
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/meta"
)

// Meta holds document metadata including title, description, and various head elements.
type Meta = meta.Meta

// MetaTag describes a <meta> element.
type MetaTag = html.MetaTag

// LinkTag describes a <link> element.
type LinkTag = html.LinkTag

// ScriptTag describes a <script> element.
type ScriptTag = html.ScriptTag

// UseMetaTags sets document metadata (title, meta tags, links, scripts) for the current page.
// Example: UseMetaTags(ctx, &Meta{Title: "Page Title", Description: "Page description"})
func UseMetaTags(ctx Ctx, m *Meta) {
	meta.UseMetaTags(ctx, m)
}
