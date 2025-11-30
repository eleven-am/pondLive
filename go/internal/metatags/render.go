package metatags

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

var Render = runtime.Component(func(ctx *runtime.Ctx, children []work.Node) work.Node {
	controller := metaCtx.UseContextValue(ctx)
	metaData := controller.Get()

	items := make([]work.Node, 0)

	if metaData.Title != "" {
		items = append(items, &work.Element{
			Tag:      "title",
			Children: []work.Node{&work.Text{Value: metaData.Title}},
		})
	}

	if metaData.Description != "" {
		descMeta := MetaTags(MetaTag{
			Name:    "description",
			Content: metaData.Description,
		})
		items = append(items, descMeta...)
	}

	items = append(items, MetaTags(metaData.Meta...)...)
	items = append(items, LinkTags(metaData.Links...)...)
	items = append(items, ScriptTags(metaData.Scripts...)...)

	return &work.Fragment{Children: items}
})
