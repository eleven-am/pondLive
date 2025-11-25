package metatags

import (
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Head renders the <head> element with meta tags from context.
func Head(ctx *runtime.Ctx) work.Node {
	controller := metaCtx.UseContextValue(ctx)
	metaData := controller.Get()

	items := make([]html.Item, 0)

	if metaData.Title != "" {
		items = append(items, html.TitleEl(html.Text(metaData.Title)))
	}

	if metaData.Description != "" {
		descMeta := html.MetaTags(html.MetaTag{
			Name:    "description",
			Content: metaData.Description,
		})
		for _, node := range descMeta {
			items = append(items, node.(html.Item))
		}
	}

	for _, node := range html.MetaTags(metaData.Meta...) {
		items = append(items, node.(html.Item))
	}

	for _, node := range html.LinkTags(metaData.Links...) {
		items = append(items, node.(html.Item))
	}

	for _, node := range html.ScriptTags(metaData.Scripts...) {
		items = append(items, node.(html.Item))
	}

	return html.Head(items...)
}
