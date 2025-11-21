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
// Useful for SEO, Open Graph tags, and dynamic page titles in single-page applications.
//
// Example - Set page title and description:
//
//	import "github.com/eleven-am/pondlive/go/pkg/live"
//
//	func BlogPost(ctx live.Ctx) live.Node {
//	    post, _ := live.UseState(ctx, Post{})
//	    postID := live.UseParam(ctx, "postID")
//
//	    live.UseEffect(ctx, func() live.Cleanup {
//	        p, err := fetchPost(postID)
//	        if err == nil {
//	            post(p)
//	        }
//	        return nil
//	    }, postID)
//
//	    live.UseMetaTags(ctx, &live.Meta{
//	        Title:       post().Title + " - My Blog",
//	        Description: post().Summary,
//	    })
//
//	    return h.Article(
//	        h.H1(h.Text(post().Title)),
//	        h.P(h.Text(post().Content)),
//	    )
//	}
//
// Example - Full metadata with Open Graph tags:
//
//	func ProductPage(ctx live.Ctx) live.Node {
//	    product, _ := live.UseState(ctx, Product{})
//
//	    live.UseMetaTags(ctx, &live.Meta{
//	        Title:       product().Name,
//	        Description: product().Description,
//	        Meta: []live.MetaTag{
//	            {Property: "og:title", Content: product().Name},
//	            {Property: "og:description", Content: product().Description},
//	            {Property: "og:image", Content: product().ImageURL},
//	            {Property: "og:type", Content: "product"},
//	        },
//	        Links: []live.LinkTag{
//	            {Rel: "canonical", Href: "https://example.com/products/" + product().ID},
//	        },
//	    })
//
//	    return h.Div(/* render product */)
//	}
//
// Example - Adding scripts and stylesheets:
//
//	func Dashboard(ctx live.Ctx) live.Node {
//	    live.UseMetaTags(ctx, &live.Meta{
//	        Title: "Dashboard",
//	        Links: []live.LinkTag{
//	            {Rel: "stylesheet", Href: "/css/dashboard.css"},
//	        },
//	        Scripts: []live.ScriptTag{
//	            {Src: "/js/charts.js", Defer: true},
//	            {Inner: "console.log('Dashboard loaded')", Type: "text/javascript"},
//	        },
//	    })
//
//	    return h.Div(/* dashboard content */)
//	}
//
// Note: This hook must be used within a component tree that has meta.Provider at the root.
func UseMetaTags(ctx Ctx, m *Meta) {
	meta.UseMetaTags(ctx, m)
}
