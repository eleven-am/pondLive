package meta

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/pkg/live"
)

type Meta struct {
	Title       string
	Description string
	Meta        []MetaTag
	Links       []LinkTag
	Scripts     []ScriptTag
}

type metaController struct {
	get func() *Meta
	set func(*Meta)
}

var defaultMeta = &Meta{
	Title:       "PondLive Application",
	Description: "A PondLive application",
}

var metaTagsCtx = runtime.CreateContext[*metaController](nil)

// UseMetaTags sets document metadata (title, meta tags, links, scripts) for the current page.
// Useful for SEO, Open Graph tags, and dynamic page titles in single-page applications.
//
// Example - Set page title and description:
//
//	import "github.com/eleven-am/pondlive/go/pkg/live/meta"
//
//	func BlogPost(ctx live.Ctx) h.Node {
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
//	    meta.UseMetaTags(ctx, &meta.Meta{
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
//	func ProductPage(ctx live.Ctx) h.Node {
//	    product, _ := live.UseState(ctx, Product{})
//
//	    meta.UseMetaTags(ctx, &meta.Meta{
//	        Title:       product().Name,
//	        Description: product().Description,
//	        Meta: []meta.MetaTag{
//	            {Property: "og:title", Content: product().Name},
//	            {Property: "og:description", Content: product().Description},
//	            {Property: "og:image", Content: product().ImageURL},
//	            {Property: "og:type", Content: "product"},
//	        },
//	        Links: []meta.LinkTag{
//	            {Rel: "canonical", Href: "https://example.com/products/" + product().ID},
//	        },
//	    })
//
//	    return h.Div(/* render product */)
//	}
//
// Example - Adding scripts and stylesheets:
//
//	func Dashboard(ctx live.Ctx) h.Node {
//	    meta.UseMetaTags(ctx, &meta.Meta{
//	        Title: "Dashboard",
//	        Links: []meta.LinkTag{
//	            {Rel: "stylesheet", Href: "/css/dashboard.css"},
//	        },
//	        Scripts: []meta.ScriptTag{
//	            {Src: "/js/charts.js", Defer: true},
//	            {Inner: "console.log('Dashboard loaded')", Type: "text/javascript"},
//	        },
//	    })
//
//	    return h.Div(/* dashboard content */)
//	}
//
// Note: This hook must be used within a meta.Provider component tree.
func UseMetaTags(ctx live.Ctx, meta *Meta) {
	controller := metaTagsCtx.Use(ctx)
	if controller != nil {
		controller.set(meta)
	}
}
