package meta

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Head renders the <head> element with meta tags from context.
func Head(ctx runtime.Ctx) *dom.StructuredNode {
	controller := metaCtx.Use(ctx)
	metaData := controller.Get()

	items := make([]dom.Item, 0)

	if metaData.Title != "" {
		items = append(items, dom.El(html.HTMLTitleElement{}, dom.TextNode(metaData.Title)))
	}

	if metaData.Description != "" {
		descMeta := html.MetaTags(html.MetaTag{
			Name:    "description",
			Content: metaData.Description,
		})
		for _, node := range descMeta {
			items = append(items, node)
		}
	}

	for _, node := range html.MetaTags(metaData.Meta...) {
		items = append(items, node)
	}

	for _, node := range html.LinkTags(metaData.Links...) {
		items = append(items, node)
	}

	for _, node := range html.ScriptTags(metaData.Scripts...) {
		items = append(items, node)
	}

	return dom.El(html.HTMLHeadElement{}, items...)
}
