package metatags

import (
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

var Render = html.Component(func(ctx *runtime.Ctx, children []work.Node) work.Node {
	controller := metaCtx.UseContextValue(ctx)
	metaData := controller.Get()

	items := make([]work.Node, 0)

	if metaData.Title != "" {
		items = append(items, html.TitleEl(html.Text(metaData.Title)))
	}

	if metaData.Description != "" {
		descMeta := html.MetaTags(html.MetaTag{
			Name:    "description",
			Content: metaData.Description,
		})
		items = append(items, descMeta...)
	}

	items = append(items, html.MetaTags(metaData.Meta...)...)
	items = append(items, html.LinkTags(metaData.Links...)...)
	items = append(items, html.ScriptTags(metaData.Scripts...)...)

	return &work.Fragment{Children: items}
})
