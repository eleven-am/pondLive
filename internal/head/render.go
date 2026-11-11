package head

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type ElementProps struct {
	Children []work.Item
}

var HTML = runtime.PropsComponent(func(ctx *runtime.Ctx, props ElementProps) work.Node {
	reg := UseRegistry(ctx)
	items := []work.Item{
		work.Attribute("lang", reg.lang, work.OmitEmpty()),
		work.Attribute("dir", reg.dir, work.OmitEmpty()),
		work.Class(reg.htmlClass),
	}
	return work.BuildElement("html", append(items, props.Children...)...)
})

var Body = runtime.PropsComponent(func(ctx *runtime.Ctx, props ElementProps) work.Node {
	reg := UseRegistry(ctx)
	items := []work.Item{work.Class(reg.bodyClass)}
	return work.BuildElement("body", append(items, props.Children...)...)
})

var Render = runtime.Component(func(ctx *runtime.Ctx) work.Node {
	reg := UseRegistry(ctx)
	hasDescription := runtime.UseComputed(ctx, func() bool { return reg.description.Get() != "" })

	return work.NewFragment(
		work.BuildElement("title", work.TextOf(reg.title)),
		runtime.Show(hasDescription, func() work.Node {
			return work.BuildElement("meta", work.Attribute("name", "description"), work.Attribute("content", reg.description))
		}),
		runtime.Show(reg.hasIcon, func() work.Node {
			return work.BuildElement("link", work.Attribute("rel", "icon"), work.Attribute("type", "image/png"), work.Attribute("sizes", "32x32"), work.Attribute("href", reg.png32URL))
		}),
		runtime.Show(reg.hasIcon, func() work.Node {
			return work.BuildElement("link", work.Attribute("rel", "apple-touch-icon"), work.Attribute("href", reg.png180URL))
		}),
		runtime.For(runtime.Items(reg.meta, MetaTag.signature), runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[MetaTag]) work.Node {
			return item.Peek().ToNode()
		})),
		runtime.For(runtime.Items(reg.links, LinkTag.signature), runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[LinkTag]) work.Node {
			return item.Peek().ToNode()
		})),
		runtime.For(runtime.Items(reg.styles, StyleTag.signature), runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[StyleTag]) work.Node {
			return item.Peek().ToNode()
		})),
		runtime.For(runtime.Items(reg.scripts, ScriptTag.signature), runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[ScriptTag]) work.Node {
			return item.Peek().ToNode()
		})),
	)
})
