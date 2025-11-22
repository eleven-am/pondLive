package meta

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

var script = live.PropsComponent[ScriptTag](func(ctx live.Ctx, prop ScriptTag, _ []h.Item) h.Node {
	return h.Script(
		h.If(prop.Src != "", h.Attr("src", prop.Src)),
		h.If(prop.Type != "", h.Attr("type", prop.Type)),
		h.If(prop.Async, h.Attr("async", "true")),
		h.If(prop.Defer, h.Attr("defer", "true")),
		h.If(prop.Module, h.Attr("type", "module")),
		h.If(prop.NoModule, h.Attr("nomodule", "true")),
		h.If(prop.CrossOrigin != "", h.Attr("crossorigin", prop.CrossOrigin)),
		h.If(prop.Integrity != "", h.Attr("integrity", prop.Integrity)),
		h.If(prop.ReferrerPolicy != "", h.Attr("referrerpolicy", prop.ReferrerPolicy)),
		h.If(prop.Nonce != "", h.Attr("nonce", prop.Nonce)),
		h.If(prop.Inner != "", h.Text(prop.Inner)),
	)
})

var link = live.PropsComponent[LinkTag](func(ctx live.Ctx, prop LinkTag, _ []h.Item) h.Node {
	return h.Link(
		h.If(prop.Rel != "", h.Attr("rel", prop.Rel)),
		h.If(prop.Href != "", h.Attr("href", prop.Href)),
		h.If(prop.Type != "", h.Attr("type", prop.Type)),
		h.If(prop.As != "", h.Attr("as", prop.As)),
		h.If(prop.Media != "", h.Attr("media", prop.Media)),
		h.If(prop.HrefLang != "", h.Attr("hreflang", prop.HrefLang)),
		h.If(prop.Title != "", h.Attr("title", prop.Title)),
		h.If(prop.CrossOrigin != "", h.Attr("crossorigin", prop.CrossOrigin)),
		h.If(prop.Integrity != "", h.Attr("integrity", prop.Integrity)),
		h.If(prop.ReferrerPolicy != "", h.Attr("referrerpolicy", prop.ReferrerPolicy)),
		h.If(prop.Sizes != "", h.Attr("sizes", prop.Sizes)),
	)
})

var meta = live.PropsComponent[MetaTag](func(ctx live.Ctx, prop MetaTag, _ []h.Item) h.Node {
	return h.Meta(
		h.If(prop.Name != "", h.Attr("name", prop.Name)),
		h.If(prop.Content != "", h.Attr("content", prop.Content)),
		h.If(prop.Property != "", h.Attr("property", prop.Property)),
		h.If(prop.Charset != "", h.Attr("charset", prop.Charset)),
		h.If(prop.HTTPEquiv != "", h.Attr("http-equiv", prop.HTTPEquiv)),
		h.If(prop.ItemProp != "", h.Attr("itemprop", prop.ItemProp)),
	)
})

var Head = live.Component(func(ctx live.Ctx, _ []h.Item) h.Node {
	controller := metaTagsCtx.Use(ctx)
	if controller == nil {
		return h.Head()
	}

	metaData := controller.get()

	return h.Head(
		h.Title(metaData.Title),
		h.If(metaData.Description != "", h.Meta(
			h.Attr("name", "description"),
			h.Attr("content", metaData.Description),
		)),
		h.Map(metaData.Meta, func(tag MetaTag) h.Node {
			return meta(ctx, tag)
		}),
		h.Map(metaData.Links, func(tag LinkTag) h.Node {
			return link(ctx, tag)
		}),
		h.Map(metaData.Scripts, func(tag ScriptTag) h.Node {
			return script(ctx, tag)
		}),
	)
})

func Provider(ctx live.Ctx, children ...h.Item) h.Node {
	current, setCurrent := runtime.UseState(ctx, defaultMeta)

	controller := runtime.UseMemo(ctx, func() *metaController {
		return &metaController{}
	})

	controller.get = current
	controller.set = setCurrent

	return metaTagsCtx.Provide(ctx, controller, func(ctx live.Ctx) h.Node {
		return h.Fragment(children...)
	})
}
