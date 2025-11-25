package meta2

import (
	"github.com/eleven-am/pondlive/go/internal/html2"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Head renders the <head> element with meta tags from context.
func Head(ctx *runtime2.Ctx) work.Node {
	controller := metaCtx.UseContextValue(ctx)
	metaData := controller.Get()

	items := make([]html2.Item, 0)

	if metaData.Title != "" {
		items = append(items, html2.TitleEl(html2.Text(metaData.Title)))
	}

	if metaData.Description != "" {
		descMeta := html2.MetaTags(html2.MetaTag{
			Name:    "description",
			Content: metaData.Description,
		})
		for _, node := range descMeta {
			items = append(items, node.(html2.Item))
		}
	}

	for _, node := range html2.MetaTags(metaData.Meta...) {
		items = append(items, node.(html2.Item))
	}

	for _, node := range html2.LinkTags(metaData.Links...) {
		items = append(items, node.(html2.Item))
	}

	for _, node := range html2.ScriptTags(metaData.Scripts...) {
		items = append(items, node.(html2.Item))
	}

	return html2.Head(items...)
}
